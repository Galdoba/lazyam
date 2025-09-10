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
	cycleStage_CheckLock = iota
	cycleStage_ReadCache
	cycleStage_UpdateCache
	cycleStage_ProjectProcessing
	cycleStage_TrailerProcessing
	cycleStage_Sleep
)

func Process(actx *appmodule.AppContext) cli.ActionFunc {
	return func(ctx context.Context, c *cli.Command) error {
		cfg := actx.Config
		log := actx.Log
		log.Noticef("session started")
		analitycs.StartTracker(cfg, log)
		if !c.Bool(flags.KEEP_CACHE) {
			os.Remove(cfg.Declarations.TaskCacheFile)
			os.Remove(cfg.Declarations.ProjectCacheFile)
		}

		cycle := 1
		breakinError := false
		cycleStage := cycleStage_CheckLock
		projects := projectdata.NewProjects()
		tasklist := task.NewTaskList()
		// activeTasks := make(map[string]*task.Task)
		for !breakinError {
			switch cycleStage {
			case cycleStage_CheckLock:
				if err := actionstage.CheckLock(cfg, log); err != nil {
					log.Errorf("failed to check lock: %v", err.Error())
				}
				cycleStage = cycleStage_ReadCache
			case cycleStage_ReadCache:
				cachePath := cfg.Declarations.ProjectCacheFile
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
						log.Debugf("new cache file created: %v", cachePath)
						continue
					case false:
						log.Errorf(format, errArgs...)
						return fmt.Errorf("failed to load cache data")
					}

				}
				cachePath = cfg.Declarations.TaskCacheFile
				projects = loadedMeta
				loadedTasks, err := actionstage.LoadProjectsCache(cachePath)
				switch err := err.(type) {
				case *lazyerror.LazyError:
					format, errArgs := err.FormatArgs()
					switch err.IsExpected() {
					case true:
						if err := tasklist.Save(cachePath); err != nil {
							log.Errorf("failed to create new cache file: %v", err)
							return fmt.Errorf("failed to create new cache file: %v", err)
						}
						log.Debugf("new cache file created: %v", cachePath)
						continue
					case false:
						log.Errorf(format, errArgs...)
						return fmt.Errorf("failed to load cache data")
					}

				}
				tasklist = loadedTasks
				log.Debugf("cache loaded")
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
							err := activeTask.FillMetatada(projects, actx.Log)
							switch err {
							case nil:
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
								log.Warnf("phase failed: %v", err.Error())
								continue
							}
							stageResult = 1
							activeTask.ProcessingStage = task.Phase_StartInterlaceCheck
							continue
						case task.Phase_StartInterlaceCheck:
							source := activeTask.VideoSourceName()
							if source == "" {
								break
							}

							switch strings.Contains(source, "SPO_") {
							case true:
								activeTask.IsSport = true
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
								log.Infof("interlace detection script generated: %v", check.Path())
								stageResult = 1
								activeTask.ProcessingStage = task.Phase_EvaluateInterlaceCheckResult
							case false:
								activeTask.ProcessingStage = task.Phase_EvaluateTrancecodingProcess
								activeTask.InderlaceScanned = true
								stageResult = 1
								break
							}

						case task.Phase_EvaluateInterlaceCheckResult:
							if err := activeTask.AssesInterlaceReport(cfg); err != nil {
								log.Debugf("interlace check evaluation: %v", err.Error())
								break
							}
							if activeTask.InderlaceScanned {
								log.Debugf("interlace scan completed: %v", activeTask.OUTBASE)
								log.Debugf("%v interlace=%v ratio=%v", activeTask.OUTBASE, activeTask.InterlaceDetected, activeTask.ProgressiveRatio)
								if activeTask.InterlaceDetected {
									log.Noticef("%v interlace=%v", activeTask.OUTBASE, activeTask.InterlaceDetected)
									stageResult = 1
									activeTask.ProcessingStage = task.Phase_EvaluateTrancecodingProcess
								}
								stageResult = 1
								activeTask.ProcessingStage = task.Phase_EvaluateTrancecodingProcess
							}
						case task.Phase_EvaluateTrancecodingProcess:
							log.Debugf("start trancoding phase: %v", activeTask.OUTBASE)
							source := ""
							srt := ""
							for _, v := range activeTask.MediaFiles {
								if strings.HasSuffix(v.Name, ".srt") {
									srt = v.Name
									log.Debugf("subs added: %v", srt)
								} else {
									source = v.Name
									log.Debugf("source added: %v", source)
								}
							}
							if err := moveSources(cfg, activeTask, source, srt); err != nil {
								log.Errorf("failed to move sources: %v", err.Error())
								break
							}
							source = activeTask.INBASE + "_" + source
							srt = activeTask.INBASE + "_" + srt
							template, audioSuffixes := selectTemplate(activeTask)
							transcodingProcess := &scriptkit.Script{}
							scriptPath := filepath.ToSlash(filepath.Join(cfg.Declarations.OutputDirectory, fmt.Sprintf("%v.sh", activeTask.INBASE)))

							args := []scriptkit.ScriptArgument{}
							args = append(args, scriptkit.ScriptArg("source", source))
							args = append(args, scriptkit.ScriptArg("base_with_season", activeTask.TranslitedBaseSeason()))
							args = append(args, scriptkit.ScriptArg("outbase", activeTask.OUTBASE))
							yadif := scriptkit.ScriptArg("yadif", "")
							if activeTask.InterlaceDetected {
								yadif = scriptkit.ScriptArg("yadif", "yadif,")
							}
							args = append(args, yadif)
							args = append(args, scriptkit.ScriptArg("suffix_1", audioSuffixes[0]))

							switch template {
							case scriptkit.Amedia1:
							case scriptkit.Amedia2:
								args = append(args, scriptkit.ScriptArg("suffix_2", audioSuffixes[1]))
							case scriptkit.Amedia2S:
								args = append(args, scriptkit.ScriptArg("srt", srt))
								args = append(args, scriptkit.ScriptArg("suffix_2", audioSuffixes[1]))
							}
							transcodingProcess = scriptkit.New(scriptPath, scriptkit.WithTemplate(template), scriptkit.WithArgs(args...))

							if err := transcodingProcess.CreateScriptFile(); err != nil {
								log.Errorf("failed to start interlace check: %v", err.Error())
								break
							}
							log.Infof("transcoding script generated: %v", transcodingProcess.Path())
							activeTask.ProcessingStage = task.Phase_WaitTranceCodingResult
							stageResult = 1
						}

					}
				}

				cycleStage++
			case cycleStage_TrailerProcessing:
				dir := cfg.Declarations.OutputDirectory
				fi, err := os.ReadDir(dir)
				if err != nil {
					log.Errorf("failed to check trailers: %v", err.Error())
				}
				for _, f := range fi {
					if f.IsDir() {
						continue
					}
					path := filepath.Join(dir, f.Name())
					if actionstage.IsAmediaTrailer(path) {
						err := actionstage.WriteScript(cfg, path)
						switch err {
						case nil:
							log.Infof("trailer script for %v complete", f.Name())
						default:
							log.Errorf("trailer script for %v failed", f.Name())
						}
					}
				}
				cycleStage++
			case cycleStage_Sleep:
				actionstage.Sleep(cfg.Processing.DormantMode)
				cycleStage = cycleStage_CheckLock
				cycle++
			}

		}

		return nil

	}
}

type Updater interface {
	Update(*config.Config, *logmanager.Logger) error
	Save(string) error
}

func toLinuxPath(path string) string {
	return strings.ReplaceAll(path, "//192.168.31.4/buffer/IN", "/home/pemaltynov/IN")
}

func selectTemplate(t *task.Task) (string, []string) {
	suffixes := t.Suffixes()
	keep := []string{}
	for _, suff := range suffixes {
		if !strings.Contains(suff, "RUS") && t.IsSport {
			continue
		}
		keep = append(keep, suff)
	}
	srt := false
	for _, file := range t.MediaFiles {
		if strings.Contains(file.Name, ".srt") {
			srt = true
		}
	}
	switch len(keep) {
	case 1:
		return scriptkit.Amedia1, keep
	case 2:
		if srt {
			return scriptkit.Amedia2S, keep
		}
		return scriptkit.Amedia2, keep
	}
	return "", []string{}

}

func moveSources(cfg *config.Config, t *task.Task, sources ...string) error {
	for _, src := range sources {
		if src == "" {
			continue
		}
		from := filepath.Join(t.Directory, src)
		to := filepath.Join(cfg.Declarations.OutputDirectory, t.INBASE+"_"+src)
		if err := os.Rename(from, to); err != nil {
			return err
		}
	}
	return nil
}
