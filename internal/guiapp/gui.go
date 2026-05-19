package guiapp

import (
	"fmt"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/itamaker/go-chrome-ai/internal/chrome"
	"github.com/itamaker/go-chrome-ai/internal/meta"
)

func Run() {
	a := app.New()
	w := a.NewWindow("go-chrome-ai")
	w.Resize(fyne.NewSize(1040, 640))

	// ---- header: title (left) + repo link (right)
	repoURL, _ := url.Parse(meta.RepoURL)
	header := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("go-chrome-ai", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewHyperlink(meta.RepoURL, repoURL),
		widget.NewLabel("Patch Chrome to enable Gemini / control on-device AI"),
	)

	// ---- installations card
	installs, detectErr := chrome.DetectInstallations()
	if detectErr != nil {
		dialog.ShowError(detectErr, w)
	}
	items := make([]string, 0, len(installs))
	for _, ins := range installs {
		items = append(items, fmt.Sprintf("%-7s  %s", ins.Channel, ins.UserDataPath))
	}
	if len(items) == 0 {
		items = append(items, "(no Chrome installation detected)")
	}
	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			lbl.Wrapping = fyne.TextWrapBreak
			return lbl
		},
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(items[i]) },
	)
	listScroll := container.NewVScroll(list)
	listScroll.SetMinSize(fyne.NewSize(0, 110))
	installsCard := widget.NewCard("Chrome installations", "", listScroll)

	// ---- options card: disable-AI checkbox + structured action breakdown
	disableAICheck := widget.NewCheck(
		"Disable on-device AI model download (Gemini Nano)", nil)
	disableAICheck.SetChecked(true)

	// wrapLabel creates a label that can wrap to its parent's width instead
	// of forcing it (which would prevent the HSplit divider from being dragged).
	wrapLabel := func(text string, style fyne.TextStyle) *widget.Label {
		lbl := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, style)
		lbl.Wrapping = fyne.TextWrapWord
		return lbl
	}

	flagsHeader := wrapLabel("Local flag overrides (chrome://flags)", fyne.TextStyle{Bold: true})
	policyHeader := wrapLabel("Enterprise policy (chrome://policy)", fyne.TextStyle{Bold: true})

	var flagRows, policyRows []fyne.CanvasObject
	for _, action := range chrome.DisableAIDownloadActions() {
		if action.EnterprisePolicy {
			policyRows = append(policyRows, wrapLabel("  • "+action.Label, fyne.TextStyle{}))
			if action.Detail != "" {
				policyRows = append(policyRows,
					wrapLabel("       "+action.Detail, fyne.TextStyle{Monospace: true}))
			}
			if action.PolicyNote != "" {
				policyRows = append(policyRows,
					wrapLabel("       ! "+action.PolicyNote, fyne.TextStyle{Italic: true}))
			}
			continue
		}
		flagRows = append(flagRows, wrapLabel("  • "+action.Label, fyne.TextStyle{}))
	}

	actionBox := container.NewVBox(
		flagsHeader,
		container.NewVBox(flagRows...),
		widget.NewSeparator(),
		policyHeader,
		container.NewVBox(policyRows...),
	)

	disableAICheck.OnChanged = func(checked bool) {
		if checked {
			actionBox.Show()
		} else {
			actionBox.Hide()
		}
	}
	optionsCard := widget.NewCard("Options", "",
		container.NewVBox(disableAICheck, widget.NewSeparator(), actionBox))

	// ---- run card: progress + Run button stacked, sits on the right column
	progress := widget.NewProgressBar()
	progress.SetValue(0)
	runButton := widget.NewButton("Run go-chrome-ai", nil)
	runButton.Importance = widget.HighImportance

	// ---- logs card (fills remaining space)
	logBox := widget.NewMultiLineEntry()
	logBox.SetPlaceHolder("Logs will appear here...")
	logBox.Disable()
	logBox.TextStyle = fyne.TextStyle{Monospace: true}
	logScroll := container.NewVScroll(logBox)
	logsCard := widget.NewCard("Logs", "", logScroll)

	appendLog := func(message string) {
		fyne.Do(func() {
			if logBox.Text == "" {
				logBox.SetText(message)
			} else {
				logBox.SetText(logBox.Text + "\n" + message)
			}
		})
	}

	runButton.OnTapped = func() {
		runButton.Disable()
		progress.SetValue(0)
		fyne.Do(func() { logBox.SetText("") })

		opts := chrome.Options{DisableAIModelDownload: disableAICheck.Checked}

		go func() {
			summary, runErr := chrome.Run(opts, chrome.Callbacks{
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
				fyne.Do(func() { dialog.ShowError(runErr, w) })
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
		appendLog(fmt.Sprintf("Detected %d Chrome installation(s): %s",
			len(installs), strings.Join(items, " | ")))
	}

	// ---- left column: configuration (installations + options), scrollable
	leftColumn := container.NewVScroll(container.NewVBox(
		installsCard,
		optionsCard,
	))

	// ---- right column: run controls on top, logs filling the rest
	runRow := container.NewBorder(nil, nil, nil, runButton, progress)
	runCard := widget.NewCard("Run", "", container.NewVBox(runRow))
	rightColumn := container.NewBorder(runCard, nil, nil, nil, logsCard)

	// ---- assemble: header on top, draggable HSplit underneath
	split := container.NewHSplit(leftColumn, rightColumn)
	split.SetOffset(0.5)

	content := container.NewBorder(
		container.NewVBox(header, widget.NewSeparator()),
		nil, nil, nil,
		split,
	)
	w.SetContent(content)
	w.ShowAndRun()
}
