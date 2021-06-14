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
//import "C"

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/slytomcat/llog"
	"github.com/slytomcat/yd-go/icons"
	"github.com/slytomcat/yd-go/tools"
	"github.com/slytomcat/ydisk"
	"github.com/visualfc/goqt/ui"
	"golang.org/x/text/message"
)

const about = `yd-qgo is the panel indicator for Yandex.Disk daemon.

      Version: 0.3

Copyleft 2017-2018 Sly_tom_cat (slytomcat@mail.ru)

	  License: GPL v.3

`

var (
	// Msg is the Localization printer
	Msg *message.Printer

	iconBusy  [5]*ui.QIcon
	iconIdle  *ui.QIcon
	iconPause *ui.QIcon
	iconError *ui.QIcon
)

func main() {
	ui.Run(onStart)
}

func onStart() {
	//C.savesigchld() // temporary fix for https://github.com/visualfc/goqt/issues/52
	// Initialize application and receive the application configuration
	AppCfg := tools.AppInit("yd-qgo")
	// Initialize translations
	Msg = message.NewPrinter(message.MatchLanguage("ru"))
	// Create new ydisk interface
	YD, err := ydisk.NewYDisk(AppCfg["Conf"].(string))
	if err != nil {
		llog.Critical("Fatal error:", err)
	}
	// Start daemon if it is configured
	if AppCfg["StartDaemon"].(bool) {
		err := YD.Start()
		if err != nil {
			llog.Critical("Fatal error:", err)
		}
	}
	// Initialize icon theme
	theme, ok := AppCfg["Theme"].(string)
	if !ok {
		llog.Critical("Config read error: Theme should be string")
	}

	if err := icons.PrepareIcons(); err != nil {
		llog.Critical(err)
	}

	setTheme(theme)

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
	mStartStop.OnTriggered(func() {
		switch {
		case strings.HasPrefix(mStartStop.Text(), "\u200B"):
			go YD.Start()
		case strings.HasPrefix(mStartStop.Text(), "\u2060"):
			go YD.Stop()
		}
	})
	menu.AddAction(mStartStop)
	menu.AddSeparator()
	mOutput := ui.NewActionWithTextParent("Show daemon output", menu)
	mOutput.OnTriggered(func() { notifySend(systray, "Yandex.Disk daemon output", YD.Output()) })
	menu.AddAction(mOutput)
	mPath := ui.NewActionWithTextParent("Open: "+YD.Path, menu)
	mPath.OnTriggered(func() { tools.XdgOpen(YD.Path) })
	menu.AddAction(mPath)
	mSite := ui.NewActionWithTextParent("Open YandexDisk in browser", menu)
	mSite.OnTriggered(func() { tools.XdgOpen("https://disk.yandex.com") })
	menu.AddAction(mSite)
	menu.AddSeparator()
	mHelp := ui.NewActionWithTextParent("Help", menu)
	mHelp.OnTriggered(func() { tools.XdgOpen("https://github.com/slytomcat/YD.go/wiki/FAQ&SUPPORT") })
	menu.AddAction(mHelp)
	mAbout := ui.NewActionWithTextParent("About", menu)
	mAbout.OnTriggered(func() { notifySend(systray, "About", about) })
	menu.AddAction(mAbout)
	mDon := ui.NewActionWithTextParent("Donations", menu)
	mDon.OnTriggered(func() { tools.XdgOpen("https://github.com/slytomcat/yd-go/wiki/Donations") })
	menu.AddAction(mDon)
	menu.AddSeparator()
	quit := ui.NewActionWithTextParent("Quit", menu)
	quit.OnTriggered(func() {
		if AppCfg["StopDaemon"].(bool) {
			YD.Stop()
		}
		YD.Close() // it closes Changes channel
		icons.ClearIcons()
	})
	menu.AddAction(quit)
	systray.SetContextMenu(menu)
	systray.Show()

	go func() {
		defer ui.QApplicationQuit() // request for exit from main UI loop
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
						yds.Err + " " + tools.ShortName(yds.ErrP, 30))
					mSize1.SetText(Msg.Sprintf("Used: %s/%s", yds.Used, yds.Total))
					mSize2.SetText(Msg.Sprintf("Free: %s Trash: %s", yds.Free, yds.Trash))
					if yds.ChLast { // last synchronized list changed
						smLast.Clear()
						for _, p := range yds.Last {
							short, full := tools.ShortName(p, 40), filepath.Join(YD.Path, p)
							action := ui.NewActionWithTextParent(short, smLast)
							if tools.NotExists(full) {
								action.SetDisabled(true)
							} else {
								action.OnTriggered(func() { tools.XdgOpen(full) })
							}
							smLast.AddAction(action)
						}
						mLast.SetDisabled(len(yds.Last) == 0)
						llog.Debug("Last synchronized updated L", len(yds.Last))
					}
					if yds.Stat != yds.Prev { // status changed
						// change indicator icon
						switch yds.Stat {
						case "idle":
							systray.SetIcon(iconIdle)
						case "busy", "index":
							systray.SetIcon(iconBusy[currentIcon])
							if yds.Prev != "busy" && yds.Prev != "index" {
								tick.Reset(333 * time.Millisecond)
							}
						case "none", "paused":
							systray.SetIcon(iconPause)
						default:
							systray.SetIcon(iconError)
						}
						// handle "Start"/"Stop" menu title and "Show daemon output" availability
						if yds.Stat == "none" {
							mStartStop.SetText("\u200B" + Msg.Sprint("Start daemon"))
							mOutput.SetDisabled(true)
						} else if yds.Prev == "none" || yds.Prev == "unknown" {
							mStartStop.SetText("\u2060" + Msg.Sprint("Stop daemon"))
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
				currentIcon = (currentIcon + 1) % 5
				if currentStatus == "busy" || currentStatus == "index" {
					ui.Async(func() { systray.SetIcon(iconBusy[currentIcon]) })
					tick.Reset(333 * time.Millisecond)
				}
			}
		}
	}()
}

func setTheme(theme string) {
	icons.SetTheme(theme)
	iconBusy = [5]*ui.QIcon{
		ui.NewIconWithFilename(icons.IconBusy[0]),
		ui.NewIconWithFilename(icons.IconBusy[1]),
		ui.NewIconWithFilename(icons.IconBusy[2]),
		ui.NewIconWithFilename(icons.IconBusy[3]),
		ui.NewIconWithFilename(icons.IconBusy[4]),
	}
	iconError = ui.NewIconWithFilename(icons.IconError)
	iconIdle = ui.NewIconWithFilename(icons.IconIdle)
	iconPause = ui.NewIconWithFilename(icons.IconPause)
}

func notifySend(t *ui.QSystemTrayIcon, title, message string) {
	t.ShowMessage(title, message, ui.QSystemTrayIcon_Information, 2000)
}
