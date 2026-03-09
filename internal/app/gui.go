package app

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/itamaker/go-chrome-ai/internal/chrome"
)

func RunGUI() {
	a := app.New()
	w := a.NewWindow("go-chrome-ai")
	w.Resize(fyne.NewSize(760, 560))

	installs, err := chrome.DetectInstallations()
	if err != nil {
		dialog.ShowError(err, w)
	}

	items := make([]string, 0, len(installs))
	for _, install := range installs {
		items = append(items, fmt.Sprintf("%s - %s", install.Channel, install.UserDataPath))
	}
	if len(items) == 0 {
		items = append(items, "No Chrome installation detected")
	}

	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(items[i])
		},
	)
	listScroll := container.NewVScroll(list)
	listScroll.SetMinSize(fyne.NewSize(0, 140))

	logBox := widget.NewMultiLineEntry()
	logBox.SetPlaceHolder("Logs will appear here...")
	logBox.Disable()
	logScroll := container.NewVScroll(logBox)

	progress := widget.NewProgressBar()
	progress.SetValue(0)

	appendLog := func(message string) {
		fyne.Do(func() {
			if logBox.Text == "" {
				logBox.SetText(message)
				return
			}
			logBox.SetText(logBox.Text + "\n" + message)
		})
	}

	runButton := widget.NewButton("Run go-chrome-ai", nil)
	runButton.OnTapped = func() {
		runButton.Disable()
		progress.SetValue(0)
		fyne.Do(func() { logBox.SetText("") })

		go func() {
			summary, runErr := chrome.Run(chrome.Options{}, chrome.Callbacks{
				Log: appendLog,
				Progress: func(percent int) {
					if percent < 0 {
						percent = 0
					}
					if percent > 100 {
						percent = 100
					}
					fyne.Do(func() { progress.SetValue(float64(percent) / 100.0) })
				},
			})

			fyne.Do(func() { runButton.Enable() })
			if runErr != nil {
				appendLog(fmt.Sprintf("Error: %v", runErr))
				fyne.Do(func() {
					dialog.ShowError(runErr, w)
				})
				return
			}

			appendLog(fmt.Sprintf(
				"Done. detected=%d patched=%d skipped=%d restarted=%d",
				summary.DetectedInstallations,
				summary.PatchedInstallations,
				summary.SkippedInstallations,
				summary.RestartedExecutables,
			))
			fyne.Do(func() {
				dialog.ShowInformation("Completed", "go-chrome-ai patch completed.", w)
			})
		}()
	}

	if len(installs) == 0 {
		runButton.Disable()
		appendLog("No available Chrome user-data path found.")
	} else {
		appendLog(fmt.Sprintf("Detected %d Chrome installation(s): %s", len(installs), strings.Join(items, ", ")))
	}

	top := container.NewVBox(
		widget.NewLabelWithStyle("go-chrome-ai", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Patch Local State to enable Gemini / AI features in Chrome."),
		widget.NewSeparator(),
		widget.NewLabel("Detected Chrome installations"),
		listScroll,
		progress,
		container.New(layout.NewHBoxLayout(), layout.NewSpacer(), runButton, layout.NewSpacer()),
		widget.NewLabel("Logs"),
	)
	content := container.NewBorder(top, nil, nil, nil, logScroll)

	w.SetContent(content)
	w.ShowAndRun()
}
