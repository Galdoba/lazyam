package actionstage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Galdoba/appcontext/logmanager"
	"github.com/Galdoba/lazyam/internal/appmodule/config"
)

func CheckLock(cfg *config.Config, log *logmanager.Logger) error {
	dir := cfg.Declarations.InputDirectory
	lockDetectedTime := time.Time{}
	lockDetected := false
	lockAutoremove := cfg.Processing.CycleLockAutoremove
mainLoop:
	for {
		fi, err := os.ReadDir(dir)
		if err != nil {
			log.Warnf("failed to check lock: %v")
			time.Sleep(time.Second * 5)
			continue
		}
		for _, f := range fi {
			if f.Name() == "lock" {
				switch lockDetected {
				case false:
					lockDetected = true
					lockDetectedTime = time.Now()
					log.Warnf("processing lock detected. Auto unlock in %v", timer(lockAutoremove))
					continue mainLoop
				case true:
					time.Sleep(time.Second)
					since := int(time.Since(lockDetectedTime).Seconds())
					if since >= lockAutoremove {
						if err := os.RemoveAll(filepath.Join(dir, "lock")); err != nil {
							log.Errorf("failed to unlock: %v", err)
						}
						log.Infof("processing lock removed")
						return nil
					}
					fmt.Printf("force unlock in %v                               \r", timer(lockAutoremove-since))
					continue mainLoop
				}
			}
		}
		return nil
	}
}
