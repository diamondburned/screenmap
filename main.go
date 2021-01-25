package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"

	layershell "github.com/diamondburned/gotk-layer-shell"
)

var (
	selfFork     = false
	commandAfter = ""
	monitorID    = -1
	imgType      = ""
)

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [flags...] image\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.BoolVar(&selfFork, "f", selfFork, "self-background when visible and return a PID")
	flag.StringVar(&commandAfter, "c", commandAfter, "command to run when visible")
	flag.StringVar(&imgType, "t", imgType, "image type; empty to automatically parse")
	flag.IntVar(&monitorID, "m", monitorID, "the monitor to show, -1 for primary")
	flag.Parse()

	gtk.Init(nil)
}

func fork() {
	self := exec.Command(os.Args[0],
		"-t", imgType,
		"-m", strconv.Itoa(monitorID), flag.Arg(0),
	)
	self.Env = append(os.Environ(), "NOTIFY_STDOUT=1")
	self.Stdin = os.Stdin
	self.Stderr = os.Stderr

	out, err := self.StdoutPipe()
	if err != nil {
		log.Fatalln("failed to get self stdout:", err)
	}

	if err := self.Start(); err != nil {
		log.Fatalln("failed to start self:", err)
	}

	if _, err = ioutil.ReadAll(out); err != nil {
		log.Fatalln("unexpected error waiting for self:", err)
	}

	fmt.Print(self.Process.Pid)
	os.Exit(0)
}

func main() {
	if selfFork {
		fork()
	}

	w, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)

	// Treat $NOTIFY_STDOUT specially.
	if os.Getenv("NOTIFY_STDOUT") == "1" {
		w.Connect("map", func() {
			go func() {
				log.Println("mapped")
				os.Stdout.Write([]byte{0})
				os.Stdout.Close()
			}()
		})
	}

	if commandAfter != "" {
		cmd := exec.Command("sh", "-c", commandAfter)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run the command when the window is shown.
		w.Connect("map", func() {
			go func() {
				if err := cmd.Run(); err != nil {
					log.Fatalln("command failed:", err)
				}

				w.Destroy()
			}()
		})
	}

	initWindow(w)
	w.Show()
	gtk.Main()
}

func initWindow(w *gtk.Window) {
	canLayerShell := layershell.IsSupported()

	if canLayerShell {
		layershell.InitForWindow(w)
		layershell.AutoExclusiveZoneEnable(w)
		layershell.SetAnchor(w, layershell.EdgeLeft, true)
		layershell.SetAnchor(w, layershell.EdgeTop, true)
		layershell.SetLayer(w, layershell.LayerTop)
		layershell.SetKeyboardInteractivity(w, true)
	}

	display, err := w.GetDisplay()
	if err != nil {
		log.Fatalln("failed to get display:", err)
	}

	var monitor *gdk.Monitor

	if monitorID == -1 {
		if canLayerShell {
			w.Show() // prematurely realize

			window, err := w.GetWindow()
			if err != nil {
				log.Fatalln("failed to get window:", err)
			}

			monitor, err = display.GetMonitorAtWindow(window)
			if err != nil {
				log.Fatalln("failed to get monitor:", err)
			}
		} else {
			w.Fullscreen()
		}

	} else {
		if canLayerShell {
			monitor, err = display.GetMonitor(monitorID)
			if err != nil {
				log.Fatalln("failed to get monitor:", err)
			}

			layershell.SetMonitor(w, monitor)
		} else {
			screen, _ := gdk.ScreenGetDefault()
			w.FullscreenOnMonitor(screen, monitorID)
		}
	}

	image, err := readImage(flag.Arg(0), imgType, monitor.GetGeometry())
	if err != nil {
		log.Fatalln("failed to read image:", err)
	}
	image.Show()

	w.Add(image)
	w.Connect("destroy", gtk.MainQuit)
	w.Connect("key-press-event", func(w *gtk.Window, ev *gdk.Event) bool {
		keyEvent := gdk.EventKeyNewFromEvent(ev)

		switch keyEvent.KeyVal() {
		case gdk.KEY_Escape:
			w.Destroy()
			return true
		default:
			return false
		}
	})
}
