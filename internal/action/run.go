package action

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Galdoba/appcontext/logmanager"
	"github.com/Galdoba/lazyam/internal/action/actionstage"
	"github.com/Galdoba/lazyam/internal/analitycs"
	"github.com/Galdoba/lazyam/internal/appmodule"
	"github.com/Galdoba/lazyam/internal/appmodule/config"
	"github.com/Galdoba/lazyam/internal/flags"
	"github.com/Galdoba/lazyam/internal/projectdata"
	"github.com/Galdoba/lazyam/internal/task"
	lazyerror "github.com/Galdoba/lazyam/pkg/error"
	"github.com/Galdoba/lazyam/pkg/scriptkit"
	"github.com/urfave/cli/v3"
)

const (
	cycleStage_ReadCache = iota
	cycleStage_UpdateCache
	cycleStage_ProjectProcessing
	cycleStage_Sleep
)

func Process(actx *appmodule.AppContext) cli.ActionFunc {
	return func(ctx context.Context, c *cli.Command) error {
		cfg := actx.Config
		log := actx.Log
		analitycs.StartTracker(cfg, log)
		if !c.Bool(flags.KEEP_CACHE) {
			os.Remove(cfg.Declarations.TaskCacheFile)
			os.Remove(cfg.Declarations.ProjectCacheFile)
		}

		cycle := 1
		breakinError := false
		cycleStage := cycleStage_ReadCache
		projects := projectdata.NewProjects()
		tasklist := task.NewTaskList()
		// activeTasks := make(map[string]*task.Task)
		for !breakinError {
			switch cycleStage {
			case cycleStage_ReadCache:
				log.Tracef("start cycle %v", cycle)
				cachePath := cfg.Declarations.ProjectCacheFile
				log.Tracef("load metadata cache: %v", cachePath)
				loadedMeta, err := actionstage.LoadMetadataCache(cachePath)
				switch err := err.(type) {
				case *lazyerror.LazyError:
					format, errArgs := err.FormatArgs()
					switch err.IsExpected() {
					case true:
						if err := projects.Save(cachePath); err != nil {
							log.Errorf("failed to create new cache file: %v", err)
							return fmt.Errorf("failed to create new cache file: %v", err)
						}
						log.Infof("new cache file created: %v", cachePath)
						continue
					case false:
						log.Errorf(format, errArgs...)
						return fmt.Errorf("failed to load cache data")
					}

				}
				cachePath = cfg.Declarations.TaskCacheFile
				projects = loadedMeta
				log.Tracef("load task cache: %v", cachePath)
				loadedTasks, err := actionstage.LoadProjectsCache(cachePath)
				switch err := err.(type) {
				case *lazyerror.LazyError:
					format, errArgs := err.FormatArgs()
					switch err.IsExpected() {
					case true:
						log.Warnf(format, errArgs...)
						if err := tasklist.Save(cachePath); err != nil {
							log.Errorf("failed to create new cache file: %v", err)
							return fmt.Errorf("failed to create new cache file: %v", err)
						}
						log.Infof("new cache file created: %v", cachePath)
						continue
					case false:
						log.Errorf(format, errArgs...)
						return fmt.Errorf("failed to load cache data")
					}

				}
				tasklist = loadedTasks

				cycleStage++
			case cycleStage_UpdateCache:
				dataFrom := make(map[string]Updater)
				dataFrom[cfg.Declarations.ProjectCacheFile] = projects
				dataFrom[cfg.Declarations.TaskCacheFile] = tasklist
				for _, cacheFile := range []string{
					cfg.Declarations.ProjectCacheFile,
					cfg.Declarations.TaskCacheFile,
				} {

					err := dataFrom[cacheFile].Update(cfg, log)

					switch err {
					case nil:
						if err := dataFrom[cacheFile].Save(cacheFile); err != nil {
							log.Errorf("cache saving failed: %v", err.Error())
						}
					default:
						if err.Error() != "no update needed" {
							log.Errorf("cache saving failed: %v", err.Error())
						}
					}
				}

				cycleStage++
			case cycleStage_ProjectProcessing:
				log.Infof("emulate process")
				taskDirectories, err := actionstage.ListActiveTasks(cfg)
				if err != nil {
					log.Errorf("failed to list active tasks: %v", err.Error())
				}
				if tasklist.Tasks == nil {
					tasklist.Tasks = make(map[string]*task.Task)
				}
				//add new tasks
				for _, taskKey := range taskDirectories {
					if _, ok := tasklist.Tasks[taskKey]; ok {
						continue
					} else {
						new := task.New(taskKey)
						tasklist.Tasks[taskKey] = new
						log.Infof("new task added: %v", filepath.Base(taskKey))
					}

				}
				if err := tasklist.Save(cfg.Declarations.TaskCacheFile); err != nil {
					log.Errorf("failed to save tasklist: %v", err.Error())
				}
				for key, activeTask := range tasklist.Tasks {
					done := false
					stageResult := 1
					for !done {
						if stageResult < 1 {
							done = true
						}
						if done {
							tasklist.Tasks[key] = activeTask
							if err := tasklist.Save(cfg.Declarations.TaskCacheFile); err != nil {
								log.Errorf("failed to save tasklist: %v", err.Error())
							}
							break
						}
						stageResult = 0
						switch activeTask.ProcessingStage {
						case task.Phase_SyncMeta:
							activeTask.CollectSignals()
							err := activeTask.FillMetatada(projects)
							switch err {
							case nil:
								log.Infof("metadata filled: %v", activeTask.OUTBASE)
								stageResult = 1
								activeTask.ProcessingStage = task.Phase_ScanSources
								continue
							default:
								if err.Error() == "no metadata present" {
									log.Warnf("failed to sync metadata for %v", activeTask.OUTBASE)
									log.Infof("fallback to blunt mode")
									stageResult = 1
									activeTask.ProcessingStage = task.Phase_ScanSources
									continue
								}
								log.Errorf("failed to fill %v metadata: %v", activeTask.OUTBASE, err.Error())
							}
						case task.Phase_ScanSources:
							if err := activeTask.ScanSources(); err != nil {
								log.Errorf("phase failed: %v", err.Error())
								continue
							}
							stageResult = 1
							activeTask.ProcessingStage = task.Phase_StartInterlaceCheck
							continue
						case task.Phase_StartInterlaceCheck:
							source := activeTask.VideoSourceName()
							if source == "" {
								log.Tracef("no source files for: %v", activeTask.OUTBASE)
								break
							}
							if !strings.Contains(source, "SPO") {
								activeTask.ProcessingStage = task.Phase_EvaluateTrancecodingProcess
								activeTask.InderlaceScanned = true
								stageResult = 1
							}
							log.Tracef("source for: %v", activeTask.OUTBASE)
							check := scriptkit.New(filepath.ToSlash(filepath.Join(cfg.Declarations.OutputDirectory, fmt.Sprintf("/_interlace_scan_%v.sh", activeTask.OUTBASE))),
								scriptkit.WithTemplate(scriptkit.ScanInterlace),
								scriptkit.WithArgs(
									scriptkit.ScriptArg("file", activeTask.VideoSourceName()),
									scriptkit.ScriptArg("directory", toLinuxPath(activeTask.Directory)),
								),
							)
							if err := check.CreateScriptFile(); err != nil {
								log.Errorf("failed to start interlace check: %v", err.Error())
								break
							}
							log.Infof("start interlace check: %v", check.Path())
							stageResult = 1
							activeTask.ProcessingStage = task.Phase_EvaluateInterlaceCheckResult
						case task.Phase_EvaluateInterlaceCheckResult:
							if err := activeTask.AssesInterlaceReport(); err != nil {
								log.Tracef("interlace check evaluation: %v", err.Error())
								break
							}
							if activeTask.InderlaceScanned && activeTask.InterlaceDetected {
								log.Noticef("%v interlace=%v", activeTask.OUTBASE, activeTask.InterlaceDetected)
								stageResult = 1
								activeTask.ProcessingStage = task.Phase_EvaluateTrancecodingProcess
							}
						case task.Phase_EvaluateTrancecodingProcess:
							break
						}

					}
				}

				cycleStage++
			case cycleStage_Sleep:
				actionstage.Sleep(cfg.Processing.DormantMode)
				cycleStage = cycleStage_ReadCache
				log.Tracef("end cycle %v", cycle)
				cycle++
			}

			if cycle > 3 {
				breakinError = true
			}

		}
		log.Infof("action %v ended", "run")

		return nil

	}
}

// func Run(ctx context.Context, c *cli.Command) error {
// 	fmt.Println("start run")
// 	cfg := c.Metadata["config"].(*config.Config)
// 	if cfg == nil {
// 		return fmt.Errorf("no config received")
// 	}
// 	log := c.Metadata["logger"].(*golog.Logger)
// 	if log == nil {
// 		return fmt.Errorf("no logger received")
// 	}
// 	// actx := appmodule.Initiate(declare.APP_NAME)
// 	analitycs.StartTracker(cfg, log)

// 	cycle := 1
// 	breakinError := false
// 	cycleStage := cycleStage_ReadCache
// 	projects := projectdata.NewProjects()
// 	tasklist := task.NewTaskList()
// 	// activeTasks := make(map[string]*task.Task)
// 	for !breakinError {
// 		switch cycleStage {
// 		case cycleStage_ReadCache:
// 			log.Tracef("start cycle %v", cycle)
// 			cachePath := cfg.Declarations.ProjectCacheFile
// 			log.Tracef("load metadata cache: %v", cachePath)
// 			loadedMeta, err := actionstage.LoadMetadataCache(cachePath)
// 			switch err := err.(type) {
// 			case *lazyerror.LazyError:
// 				format, errArgs := err.FormatArgs()
// 				switch err.IsExpected() {
// 				case true:
// 					log.Warnf(format, errArgs...)
// 					if err := projects.Save(cachePath); err != nil {
// 						log.Errorf("failed to create new cache file: %v", err)
// 						return fmt.Errorf("failed to create new cache file: %v", err)
// 					}
// 					log.Infof("new cache file created: %v", cachePath)
// 					continue
// 				case false:
// 					log.Errorf(format, errArgs...)
// 					return fmt.Errorf("failed to load cache data")
// 				}

// 			}
// 			cachePath = cfg.Declarations.TaskCacheFile
// 			projects = loadedMeta
// 			log.Tracef("load task cache: %v", cachePath)
// 			loadedTasks, err := actionstage.LoadProjectsCache(cachePath)
// 			switch err := err.(type) {
// 			case *lazyerror.LazyError:
// 				format, errArgs := err.FormatArgs()
// 				switch err.IsExpected() {
// 				case true:
// 					log.Warnf(format, errArgs...)
// 					if err := tasklist.Save(cachePath); err != nil {
// 						log.Errorf("failed to create new cache file: %v", err)
// 						return fmt.Errorf("failed to create new cache file: %v", err)
// 					}
// 					log.Infof("new cache file created: %v", cachePath)
// 					continue
// 				case false:
// 					log.Errorf(format, errArgs...)
// 					return fmt.Errorf("failed to load cache data")
// 				}

// 			}
// 			tasklist = loadedTasks

// 			cycleStage++
// 		case cycleStage_UpdateCache:
// 			dataFrom := make(map[string]Updater)
// 			dataFrom[cfg.Declarations.ProjectCacheFile] = projects
// 			dataFrom[cfg.Declarations.TaskCacheFile] = tasklist
// 			for _, cacheFile := range []string{
// 				cfg.Declarations.ProjectCacheFile,
// 				cfg.Declarations.TaskCacheFile,
// 			} {

// 				err := dataFrom[cacheFile].Update(cfg, log)

// 				switch err {
// 				case nil:
// 					if err := dataFrom[cacheFile].Save(cacheFile); err != nil {
// 						log.Errorf("cache saving failed: %v", err.Error())
// 					}
// 				default:
// 					if err.Error() != "no update needed" {
// 						log.Errorf("cache saving failed: %v", err.Error())
// 					}
// 				}
// 			}

// 			cycleStage++
// 		case cycleStage_ProjectProcessing:
// 			log.Infof("emulate process")
// 			taskDirectories, err := actionstage.ListActiveTasks(cfg)
// 			if err != nil {
// 				log.Errorf("failed to list active tasks: %v", err.Error())
// 			}
// 			if tasklist.Tasks == nil {
// 				tasklist.Tasks = make(map[string]*task.Task)
// 			}
// 			//add new tasks
// 			for _, taskKey := range taskDirectories {
// 				if _, ok := tasklist.Tasks[taskKey]; ok {
// 					continue
// 				} else {
// 					new := task.New(taskKey)
// 					tasklist.Tasks[taskKey] = new
// 					log.Infof("new task added: %v", filepath.Base(taskKey))
// 				}

// 			}
// 			if err := tasklist.Save(cfg.Declarations.TaskCacheFile); err != nil {
// 				log.Errorf("failed to save tasklist: %v", err.Error())
// 			}
// 			for key, activeTask := range tasklist.Tasks {
// 				done := false
// 				stageResult := 1
// 				for !done {
// 					if stageResult < 1 {
// 						done = true
// 					}
// 					if done {
// 						tasklist.Tasks[key] = activeTask
// 						if err := tasklist.Save(cfg.Declarations.TaskCacheFile); err != nil {
// 							log.Errorf("failed to save tasklist: %v", err.Error())
// 						}
// 						break
// 					}
// 					stageResult = 0
// 					switch activeTask.ProcessingStage {
// 					case task.Phase_SyncMeta:
// 						activeTask.CollectSignals()
// 						err := activeTask.FillMetatada(projects)
// 						switch err {
// 						case nil:
// 							log.Infof("metadata filled: %v", activeTask.OUTBASE)
// 							stageResult = 1
// 							activeTask.ProcessingStage = task.Phase_ScanSources
// 							continue
// 						default:
// 							if err.Error() == "no metadata present" {
// 								log.Warnf("failed to sync metadata for %v", activeTask.OUTBASE)
// 								log.Infof("fallback to blunt mode")
// 								stageResult = 1
// 								activeTask.ProcessingStage = task.Phase_ScanSources
// 								continue
// 							}
// 							log.Errorf("failed to fill %v metadata: %v", activeTask.OUTBASE, err.Error())
// 						}
// 					case task.Phase_ScanSources:
// 						if err := activeTask.ScanSources(); err != nil {
// 							log.Errorf("phase failed: %v", err.Error())
// 							continue
// 						}
// 						stageResult = 1
// 						activeTask.ProcessingStage = task.Phase_StartInterlaceCheck
// 						continue
// 					case task.Phase_StartInterlaceCheck:
// 						source := activeTask.VideoSourceName()
// 						if source == "" {
// 							log.Tracef("no source files for: %v", activeTask.OUTBASE)
// 							break
// 						}
// 						if !strings.Contains(source, "SPO") {
// 							activeTask.ProcessingStage = task.Phase_EvaluateTrancecodingProcess
// 							activeTask.InderlaceScanned = true
// 							stageResult = 1
// 						}
// 						log.Tracef("source for: %v", activeTask.OUTBASE)
// 						check := scriptkit.New(filepath.ToSlash(filepath.Join(cfg.Declarations.OutputDirectory, fmt.Sprintf("/_interlace_scan_%v.sh", activeTask.OUTBASE))),
// 							scriptkit.WithTemplate(scriptkit.ScanInterlace),
// 							scriptkit.WithArgs(
// 								scriptkit.ScriptArg("file", activeTask.VideoSourceName()),
// 								scriptkit.ScriptArg("directory", toLinuxPath(activeTask.Directory)),
// 							),
// 						)
// 						if err := check.CreateScriptFile(); err != nil {
// 							log.Errorf("failed to start interlace check: %v", err.Error())
// 							break
// 						}
// 						log.Infof("start interlace check: %v", check.Path())
// 						stageResult = 1
// 						activeTask.ProcessingStage = task.Phase_EvaluateInterlaceCheckResult
// 					case task.Phase_EvaluateInterlaceCheckResult:
// 						if err := activeTask.AssesInterlaceReport(); err != nil {
// 							log.Tracef("interlace check evaluation: %v", err.Error())
// 							break
// 						}
// 						if activeTask.InderlaceScanned && activeTask.InterlaceDetected {
// 							log.Noticef("%v interlace=%v", activeTask.OUTBASE, activeTask.InterlaceDetected)
// 							stageResult = 1
// 							activeTask.ProcessingStage = task.Phase_EvaluateTrancecodingProcess
// 						}
// 					case task.Phase_EvaluateTrancecodingProcess:
// 						break
// 					}

// 				}
// 			}

// 			cycleStage++
// 		case cycleStage_Sleep:
// 			actionstage.Sleep(cfg.Processing.DormantMode)
// 			cycleStage = cycleStage_ReadCache
// 			log.Tracef("end cycle %v", cycle)
// 			cycle++
// 		}

// 		if cycle > 3 {
// 			breakinError = true
// 		}

// 	}
// 	log.Infof("action %v ended", "run")

// 	return nil
// }

/*
Cycle:
1 Read metadata files
2 Update MetaData Cache
3 Handle Cycle Lock
4 Read Projects
5   Process projects
6 Read Feedback
7 Update Stats
8 Handle autounlock
9 Handle sleep mode

project status
1 found
2 locked
3 ready
4 interlace check
5 processing
6 done

*/

type Updater interface {
	Update(*config.Config, *logmanager.Logger) error
	Save(string) error
}

func toLinuxPath(path string) string {
	return strings.ReplaceAll(path, "//192.168.31.4/buffer/IN", "/home/pemaltynov/IN")
}
