# yd-qgo (ARCHIVED)

# NOTE: due to swich to use B-BUS for GUI implementation into [yd-go](https://github.com/slytomcat/yd-go) project this project was archived. D-BUS version of yd-go works with QT as well with other GUI libraties into various Desktop Environments. So the special QT version is not needed any more. 

## Panel indicator for Yandex-disk CLI daemon (linux/Qt)

It's just reface (Qt is used instead of GTK+) of yd-go. If you are interested in GTK+ version, visit: https://github.com/slytomcat/yd-go

Russian wiki: https://github.com/slytomcat/yd-go/wiki

I've made it as it is rather well-known task for me (I've made the similar indicator in YD-tools project in Python language: https://github.com/slytomcat/yandex-disk-indicator).

GUI (System tray icon) shows the current synchronization status by different icons. During synchronization the icon is animated. 

Desktop notifications inform user when daemon started/stopped or synchronization started/stopped.

The system try icon has menu that allows:
  - to see the current daemon status and cloud-disk properties (Used/Total/Free/Trash)
  - to see (in submenu) and open (in default program) last synchronized files 
  - to start/stop daemon
  - to see the originl output of daemon in user language
  - to open local syncronized path
  - to open cloud-disk in browser
  - to open help/support, about and donatation pages

Application has its configuration file in ~/.config/yd-qgo/default.cfg file. File is in JSON format and contain following options:
  - "Conf" - path to daemon config file (default "~/.config/yandex-disk/config.cfg"
  - "Theme" - icons theme name (default "dark", may be set to "dark" or "light")
  - "Notifications" - Display or not the desktop notifications (default true)
  - "StartDaemon" - Flag that shows should be the daemon started on app start (default true)
  - "StopDaemon" - Flag that shows should be the daemon stopped on app closure

## Get
Download source from master branch  and unzip it to the go source folder ($GOHATH/src) (it can be removed after buiding and installation).
Change current directoru to the progect folder 
    cd $GOHATH/src/yd-qgo/

## Build 
For building this prject the additional packages are requered:

Install qt-sdk (Qt 4.8)

    sudo apt-get install qt-sdk

Get package github.com/visualfc/goqt

    go get github.com/visualfc/goqt
    
Follow the package instructions (https://github.com/visualfc/goqt/blob/master/doc/install.md) to install the pacage

After goqt package installation build the indicator

    cd yd-qgo/
    ./build.bash

## Installation
Run install.bash script with root previlegies for installation.

    sudo ./install.bash


## Usage
		yd-qgo [-debug] [-config=<Path to indicator config>]

	-config string
		Path to the indicator configuration file (default "~/.config/yd.go/default.cfg")
	-debug
		Alow debugging messages to be sent to stderr


Note that yandex-disk CLI utility must be installed and connection to cloud disk mast be configured for usage the yd-qgo utility.
