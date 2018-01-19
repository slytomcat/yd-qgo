package main

/*
#include <stdio.h>
#include <signal.h>
#include <string.h>

void savesigchld() {
	struct sigaction action;
	struct sigaction old_action;
	sigaction(SIGCHLD, NULL, &action);
	action.sa_flags = action.sa_flags | SA_ONSTACK;
	sigaction(SIGCHLD, &action, &old_action);
}
*/
import "C"

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"time"

	"github.com/slytomcat/confJSON"
	"github.com/slytomcat/llog"
	"github.com/slytomcat/yd-go/ydisk"
	"github.com/visualfc/goqt/ui"
	"golang.org/x/text/message"
)

const about = `yd-qgo is the panel indicator for Yandex.Disk daemon.

      Version: Betta 0.2

Copyleft 2017-2018 Sly_tom_cat (slytomcat@mail.ru)

	  License: GPL v.3

`

var (
	// AppConfigFile stores the application configuration file path
	AppConfigFile string
	// Msg is the Localozation printer
	Msg *message.Printer

	iconBusy  [5]*ui.QIcon
	iconIdle  *ui.QIcon
	iconPause *ui.QIcon
	iconError *ui.QIcon
)

func main() {
	ui.Run(func() {
		C.savesigchld() // temporary fix for https://github.com/visualfc/goqt/issues/52
		var debug bool
		flag.BoolVar(&debug, "debug", false, "Allow debugging messages to be sent to stderr")
		flag.StringVar(&AppConfigFile, "config", "~/.config/yd-qgo/default.cfg", "Path to the indicator configuration file")
		flag.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage:\n\n\t\tyd-qgo [-debug] [-config=<Path to indicator config>]\n\n")
			flag.PrintDefaults()
		}
		flag.Parse()
		// Initialize logging facility
		llog.SetOutput(os.Stderr)
		llog.SetPrefix("")
		llog.SetFlags(log.Lshortfile | log.Lmicroseconds)
		if debug {
			llog.SetLevel(llog.DEBUG)
			llog.Info("Debugging enabled")
		} else {
			llog.SetLevel(-1)
		}
		// Initialize translations
		Msg = message.NewPrinter(message.MatchLanguage("ru"))
		// Prepare the application configuration
		// Make default app configuration values
		AppCfg := map[string]interface{}{
			"Conf":          expandHome("~/.config/yandex-disk/config.cfg"), // path to daemon config file
			"Theme":         "dark",                                         // icons theme name
			"Notifications": true,                                           // display desktop notification
			"StartDaemon":   true,                                           // start daemon on app start
			"StopDaemon":    false,                                          // stop daemon on app closure
		}
		// Check that app configuration file path exists
		AppConfigHome := expandHome("~/.config/yd-qgo")
		if notExists(AppConfigHome) {
			err := os.MkdirAll(AppConfigHome, 0766)
			if err != nil {
				llog.Critical("Can't create application configuration path:", err)
			}
		}
		// Path to app configuration file path always comes from command-line flag
		AppConfigFile = expandHome(AppConfigFile)
		llog.Debug("Configuration:", AppConfigFile)
		// Check that app configuration file exists
		if notExists(AppConfigFile) {
			//Create and save new configuration file with default values
			confJSON.Save(AppConfigFile, AppCfg)
		} else {
			// Read app configuration file
			confJSON.Load(AppConfigFile, &AppCfg)
		}
		// Create new ydisk interface
		YD := ydisk.NewYDisk(AppCfg["Conf"].(string))
		// Start daemon if it is configured
		if AppCfg["StartDaemon"].(bool) {
			YD.Start()
		}
		// Initialize icon theme
		setTheme("/usr/share/yd-qgo/icons", AppCfg["Theme"].(string))
		systray := ui.NewSystemTrayIcon()
		systray.SetIcon(iconPause)
		menu := ui.NewMenu()

		mStatus := ui.NewActionWithTextParent("Status: unknown", menu)
		mStatus.SetDisabled(true)
		menu.AddAction(mStatus)
		mSize1 := ui.NewActionWithTextParent("", menu)
		mSize1.SetDisabled(true)
		menu.AddAction(mSize1)
		mSize2 := ui.NewActionWithTextParent("", menu)
		mSize2.SetDisabled(true)
		menu.AddAction(mSize2)
		menu.AddSeparator()
		mLast := ui.NewActionWithTextParent("Last synchronized", menu)
		smLast := ui.NewMenu()
		mLast.SetMenu(smLast)
		mLast.SetDisabled(true)
		menu.AddAction(mLast)
		menu.AddSeparator()
		mStartStop := ui.NewActionWithTextParent("", menu)
		mStartStop.SetDisabled(true)
		mStartStop.OnTriggered(func() {
			switch mStartStop.Text() {
			case "Start": // start
				go YD.Start()
			case "Stop": // stop
				go YD.Stop()
			}
		})
		menu.AddAction(mStartStop)
		menu.AddSeparator()
		mOutput := ui.NewActionWithTextParent("Show daemon output", menu)
		mOutput.OnTriggered(func() { notifySend(systray, "Yandex.Disk daemon output", YD.Output()) })
		menu.AddAction(mOutput)
		mPath := ui.NewActionWithTextParent("Open: "+YD.Path, menu)
		mPath.OnTriggered(func() { xdgOpen(YD.Path) })
		menu.AddAction(mPath)
		mSite := ui.NewActionWithTextParent("Open YandexDisk in browser", menu)
		mSite.OnTriggered(func() { xdgOpen("https://disk.yandex.com") })
		menu.AddAction(mSite)
		menu.AddSeparator()
		mHelp := ui.NewActionWithTextParent("Help", menu)
		mHelp.OnTriggered(func() { xdgOpen("https://github.com/slytomcat/YD.go/wiki/FAQ&SUPPORT") })
		menu.AddAction(mHelp)
		mAbout := ui.NewActionWithTextParent("About", menu)
		mAbout.OnTriggered(func() { notifySend(systray, "About", about) })
		menu.AddAction(mAbout)
		mDon := ui.NewActionWithTextParent("Donations", menu)
		mDon.OnTriggered(func() { xdgOpen("https://github.com/slytomcat/yd-go/wiki/Donats") })
		menu.AddAction(mDon)
		menu.AddSeparator()
		quit := ui.NewActionWithTextParent("Quit", menu)
		quit.OnTriggered(func() {
			if AppCfg["StopDaemon"].(bool) {
				YD.Stop()
			}
			YD.Close() // it closes Changes channel
			ui.QApplicationQuit()
		})
		menu.AddAction(quit)
		systray.SetContextMenu(menu)
		systray.Show()

		////// ui.QApplicationPostEvent(systray, nil)
		go func() {
			defer os.Exit(0) // request for exit from systray main loop (gtk.main())
			llog.Debug("Changes handler started")
			defer llog.Debug("Changes handler exited.")
			// Prepare the staff for icon animation
			currentIcon := 0
			tick := time.NewTimer(333 * time.Millisecond)
			defer tick.Stop()
			currentStatus := ""
			for {
				select {
				case yds, ok := <-YD.Changes: // YD changed status event
					if !ok { // as Changes channel closed - exit
						return
					}
					llog.Debug("Change received")
					currentStatus = yds.Stat
					ui.Async(func() {
						mStatus.SetText(Msg.Sprint("Status: ") + Msg.Sprint(yds.Stat) + " " + yds.Prog +
							yds.Err + " " + shortName(yds.ErrP, 30))
						mSize1.SetText(Msg.Sprintf("Used: %s/%s", yds.Used, yds.Total))
						mSize2.SetText(Msg.Sprintf("Free: %s Trash: %s", yds.Free, yds.Trash))
						if yds.ChLast { // last synchronized list changed
							smLast.Clear()
							for _, p := range yds.Last {
								short, full := shortName(p, 40), filepath.Join(YD.Path, p)
								action := ui.NewActionWithTextParent(short, smLast)
								if notExists(full) {
									action.SetDisabled(true)
								} else {
									action.OnTriggered(func() { xdgOpen(full) })
								}
								smLast.AddAction(action)
							}
							mLast.SetDisabled(len(yds.Last) == 0)
							mLast.SetMenu(smLast)
							llog.Debug("Last synchronized updated L", len(yds.Last))
						}
						if yds.Stat != yds.Prev { // status changed
							// change indicator icon
							switch yds.Stat {
							case "idle":
								systray.SetIcon(iconIdle)
							case "busy", "index":
								systray.SetIcon(iconBusy[currentIcon])
								tick.Reset(333 * time.Millisecond)
							case "none", "paused":
								systray.SetIcon(iconPause)
							default:
								systray.SetIcon(iconError)
							}
							// handle "Start"/"Stop" menu title and "Show daemon output" availability 
							if yds.Stat == "none" {
								mStartStop.SetText("Start")
								mStartStop.SetDisabled(false)
								mOutput.SetDisabled(true)
							} else if mStartStop.Text() != "Stop" {
								mStartStop.SetText("Stop")
								mStartStop.SetDisabled(false)
								mOutput.SetDisabled(false)
							}
							// handle notifications
							if AppCfg["Notifications"].(bool) {
								switch {
								case yds.Stat == "none" && yds.Prev != "unknown":
									notifySend(
										systray,
										Msg.Sprint("Yandex.Disk"),
										Msg.Sprint("Daemon stopped"))
								case yds.Prev == "none":
									notifySend(
										systray,
										Msg.Sprint("Yandex.Disk"),
										Msg.Sprint("Daemon started"))
								case (yds.Stat == "busy" || yds.Stat == "index") &&
									(yds.Prev != "busy" && yds.Prev != "index"):
									notifySend(
										systray,
										Msg.Sprint("Yandex.Disk"),
										Msg.Sprint("Synchronization started"))
								case (yds.Stat == "idle" || yds.Stat == "error") &&
									(yds.Prev == "busy" || yds.Prev == "index"):
									notifySend(
										systray,
										Msg.Sprint("Yandex.Disk"),
										Msg.Sprint("Synchronization finished"))
								}
							}
						}
						systray.Show()
					})
					llog.Debug("Change handled")
				case <-tick.C: //  timer event
					currentIcon++
					currentIcon %= 5
					if currentStatus == "busy" || currentStatus == "index" {
						ui.Async(func() { systray.SetIcon(iconBusy[currentIcon]) })
						tick.Reset(333 * time.Millisecond)
					}
				}
			}
		}()

	})
}

func setTheme(appHome, theme string) {
	themePath := path.Join(appHome, theme)
	iconBusy = [5]*ui.QIcon{
		ui.NewIconWithFilename(path.Join(themePath, "busy1.png")),
		ui.NewIconWithFilename(path.Join(themePath, "busy2.png")),
		ui.NewIconWithFilename(path.Join(themePath, "busy3.png")),
		ui.NewIconWithFilename(path.Join(themePath, "busy4.png")),
		ui.NewIconWithFilename(path.Join(themePath, "busy5.png")),
	}
	iconError = ui.NewIconWithFilename(path.Join(themePath, "error.png"))
	iconIdle = ui.NewIconWithFilename(path.Join(themePath, "idle.png"))
	iconPause = ui.NewIconWithFilename(path.Join(themePath, "pause.png"))
}

func notifySend(t *ui.QSystemTrayIcon, title, message string) {
	t.ShowMessage(title, message, ui.QSystemTrayIcon_Information, 2000)
}

// shortName returns the shorten version of its first parameter. The second parameter specifies
// the maximum number of symbols (runes) in returned string.
func shortName(s string, l int) string {
	r := []rune(s)
	lr := len(r)
	if lr > l {
		b := (l - 3) / 2
		e := b
		if b+e+3 < l {
			e++
		}
		return string(r[:b]) + "..." + string(r[lr-e:])
	}
	return s
}

// notExists returns true when specified path does not exists
func notExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsNotExist(err)
	}
	return false
}

func expandHome(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	usr, err := user.Current()
	if err != nil {
		llog.Critical("Can't get current user profile:", err)
	}
	return filepath.Join(usr.HomeDir, path[1:])
}

func xdgOpen(uri string) {
	err := exec.Command("xdg-open", uri).Start()
	if err != nil {
		llog.Error(err)
	}
}
