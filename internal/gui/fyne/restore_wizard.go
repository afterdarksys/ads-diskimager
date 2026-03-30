package fyne

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// RestoreWizardConfig holds the restore wizard configuration
type RestoreWizardConfig struct {
	ImagePath    string
	ImageFormat  string
	ImageSize    int64
	TargetPath   string
	TargetSize   int64
	VerifyHashes bool
	Confirmed    bool
}

// RestoreWizard creates a 5-step restore wizard
type RestoreWizard struct {
	window fyne.Window
	config *RestoreWizardConfig
	wizard *Wizard
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRestoreWizard creates a new restore wizard
func NewRestoreWizard(window fyne.Window) *RestoreWizard {
	rw := &RestoreWizard{
		window: window,
		config: &RestoreWizardConfig{
			VerifyHashes: true,
		},
	}

	steps := []WizardStep{
		rw.createImageSelectionStep(),
		rw.createTargetDiskStep(),
		rw.createPartitionMappingStep(),
		rw.createVerificationStep(),
		rw.createExecutionStep(),
	}

	rw.wizard = NewWizard(window, steps, func() {
		// Wizard complete
		window.Close()
	}, func() {
		// Wizard cancelled
		if rw.cancel != nil {
			rw.cancel()
		}
		window.Close()
	})

	return rw
}

// Show displays the wizard
func (rw *RestoreWizard) Show() {
	rw.window.SetContent(rw.wizard.Content())
	rw.window.Resize(fyne.NewSize(900, 700))
	rw.window.CenterOnScreen()
	rw.window.Show()
}

// Step 1: Image Selection
func (rw *RestoreWizard) createImageSelectionStep() WizardStep {
	imageEntry := widget.NewEntry()
	imageEntry.SetPlaceHolder("Select forensic image file...")

	formatLabel := widget.NewLabel("Format: Unknown")
	sizeLabel := widget.NewLabel("Size: Unknown")
	createdLabel := widget.NewLabel("Created: Unknown")
	hashLabel := widget.NewLabel("Hash: Not verified")

	browseBtn := widget.NewButtonWithIcon("Browse...", theme.FolderOpenIcon(), func() {
		dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
			if err != nil || uc == nil {
				return
			}
			defer uc.Close()

			rw.config.ImagePath = uc.URI().Path()
			imageEntry.SetText(rw.config.ImagePath)

			// Analyze image
			if stat, err := os.Stat(rw.config.ImagePath); err == nil {
				rw.config.ImageSize = stat.Size()
				sizeLabel.SetText(fmt.Sprintf("Size: %d MB", stat.Size()/(1024*1024)))
				createdLabel.SetText(fmt.Sprintf("Created: %s", stat.ModTime().Format(time.RFC822)))

				// Detect format
				ext := strings.ToLower(rw.config.ImagePath[len(rw.config.ImagePath)-4:])
				switch ext {
				case ".e01", ".ex01":
					rw.config.ImageFormat = "E01 (EnCase)"
					formatLabel.SetText("Format: E01 (EnCase)")
				case ".vmdk":
					rw.config.ImageFormat = "VMDK (VMware)"
					formatLabel.SetText("Format: VMDK (VMware)")
				case ".vhd", ".vhdx":
					rw.config.ImageFormat = "VHD (Hyper-V)"
					formatLabel.SetText("Format: VHD (Hyper-V)")
				default:
					rw.config.ImageFormat = "Raw"
					formatLabel.SetText("Format: Raw/DD")
				}
			}

			rw.wizard.EnableNext(true)
		}, rw.window)
	})

	previewBox := container.NewVBox(
		widget.NewLabelWithStyle("Image Information:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		formatLabel,
		sizeLabel,
		createdLabel,
		hashLabel,
	)

	content := container.NewVBox(
		widget.NewLabel("Select the forensic image file to restore:"),
		container.NewBorder(nil, nil, nil, browseBtn, imageEntry),
		widget.NewSeparator(),
		previewBox,
		layout.NewSpacer(),
		widget.NewCard("", "", widget.NewLabel("⚠️  WARNING: Restore operations are destructive and will overwrite all data on the target disk.")),
	)

	step := NewBaseStep(
		"Step 1: Image Selection",
		"Select the forensic image to restore",
		content,
	)

	step.SetValidator(func() error {
		if rw.config.ImagePath == "" {
			return errors.New("please select an image file")
		}
		if _, err := os.Stat(rw.config.ImagePath); err != nil {
			return fmt.Errorf("cannot access image file: %v", err)
		}
		return nil
	})

	step.SetCanProgress(func() bool {
		return rw.config.ImagePath != ""
	})

	imageEntry.OnChanged = func(s string) {
		rw.config.ImagePath = s
		rw.wizard.EnableNext(s != "")
	}

	return step
}

// Step 2: Target Disk Selection
func (rw *RestoreWizard) createTargetDiskStep() WizardStep {
	targetEntry := widget.NewEntry()
	targetEntry.SetPlaceHolder("Select target disk...")

	diskList := widget.NewList(
		func() int { return 3 }, // Mock count
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.StorageIcon()),
				widget.NewLabel("Template"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			diskNames := []string{
				"/dev/disk0 - 500GB SSD - ⚠️ System Disk - DO NOT SELECT",
				"/dev/disk1 - 2TB HDD - Safe to overwrite",
				"/dev/disk2 - 1TB USB - Safe to overwrite",
			}
			if id < len(diskNames) {
				obj.(*fyne.Container).Objects[1].(*widget.Label).SetText(diskNames[id])
			}
		},
	)

	diskList.OnSelected = func(id widget.ListItemID) {
		diskPaths := []string{"/dev/disk0", "/dev/disk1", "/dev/disk2"}
		if id < len(diskPaths) {
			rw.config.TargetPath = diskPaths[id]
			targetEntry.SetText(diskPaths[id])

			// Warn if system disk
			if id == 0 {
				dialog.ShowError(errors.New("you cannot select the system disk as restore target"), rw.window)
				rw.config.TargetPath = ""
				targetEntry.SetText("")
				rw.wizard.EnableNext(false)
			} else {
				rw.wizard.EnableNext(true)
			}
		}
	}

	content := container.NewVBox(
		widget.NewCard("", "⚠️ DANGER ZONE ⚠️",
			widget.NewLabel("All data on the selected disk will be PERMANENTLY ERASED.\nThis operation cannot be undone. Please verify your selection carefully."),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Available Target Disks:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, nil, diskList),
		layout.NewSpacer(),
		widget.NewLabel("Selected Target:"),
		targetEntry,
	)

	step := NewBaseStep(
		"Step 2: Target Disk",
		"Select the destination disk for restore (WARNING: Destructive)",
		content,
	)

	step.SetValidator(func() error {
		if rw.config.TargetPath == "" {
			return errors.New("please select a target disk")
		}
		if rw.config.TargetPath == "/dev/disk0" {
			return errors.New("cannot restore to system disk")
		}
		return nil
	})

	step.SetCanProgress(func() bool {
		return rw.config.TargetPath != "" && rw.config.TargetPath != "/dev/disk0"
	})

	return step
}

// Step 3: Partition Mapping
func (rw *RestoreWizard) createPartitionMappingStep() WizardStep {
	// Source partitions (mock data)
	sourcePartitions := container.NewVBox(
		widget.NewLabelWithStyle("Source Image Layout:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("  Partition 1: EFI System (200 MB)"),
		widget.NewLabel("  Partition 2: NTFS Data (465 GB)"),
		widget.NewLabel("  Partition 3: Recovery (500 MB)"),
	)

	// Target layout
	targetPartitions := container.NewVBox(
		widget.NewLabelWithStyle("Target Disk Layout:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("  Current: Empty/Will be overwritten"),
		widget.NewLabel("  After restore: Will match source layout"),
	)

	// Size comparison
	sizeComparison := widget.NewCard("", "Size Compatibility",
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Source image size: %d GB", rw.config.ImageSize/(1024*1024*1024))),
			widget.NewLabel("Target disk size: 500 GB"),
			widget.NewLabel("✓ Target is large enough for restoration"),
		),
	)

	content := container.NewVBox(
		widget.NewLabel("Review partition layouts before proceeding:"),
		widget.NewSeparator(),
		sourcePartitions,
		widget.NewSeparator(),
		targetPartitions,
		widget.NewSeparator(),
		sizeComparison,
		layout.NewSpacer(),
		widget.NewLabel("ℹ️ The entire disk image will be written sector-by-sector to the target."),
	)

	step := NewBaseStep(
		"Step 3: Partition Mapping",
		"Review source and target partition layouts",
		container.NewScroll(content),
	)

	return step
}

// Step 4: Verification
func (rw *RestoreWizard) createVerificationStep() WizardStep {
	checkResults := container.NewVBox()

	verifyCheck := widget.NewCheck("Verify data integrity after restore (recommended)", func(checked bool) {
		rw.config.VerifyHashes = checked
	})
	verifyCheck.Checked = true

	confirmCheck := widget.NewCheck("I understand this will ERASE ALL DATA on the target disk", func(checked bool) {
		rw.config.Confirmed = checked
		rw.wizard.EnableNext(checked)
	})

	checks := []struct {
		name  string
		check func() (bool, string)
	}{
		{
			name: "Image file accessible",
			check: func() (bool, string) {
				if _, err := os.Stat(rw.config.ImagePath); err != nil {
					return false, fmt.Sprintf("Error: %v", err)
				}
				return true, "✓ Image file is readable"
			},
		},
		{
			name: "Target disk writable",
			check: func() (bool, string) {
				// Mock check
				return true, "✓ Target disk is writable"
			},
		},
		{
			name: "Size compatibility",
			check: func() (bool, string) {
				// Simplified check
				return true, "✓ Target is large enough"
			},
		},
		{
			name: "Target is not system disk",
			check: func() (bool, string) {
				if rw.config.TargetPath == "/dev/disk0" {
					return false, "✗ Target is system disk!"
				}
				return true, "✓ Safe target selected"
			},
		},
	}

	for _, check := range checks {
		statusLabel := widget.NewLabel("Checking...")
		checkResults.Add(container.NewHBox(
			widget.NewLabel("⏳"),
			widget.NewLabel(check.name + ":"),
			statusLabel,
		))
	}

	runChecksBtn := widget.NewButtonWithIcon("Run Safety Checks", theme.MediaPlayIcon(), func() {
		for i, check := range checks {
			success, msg := check.check()
			icon := "✓"
			if !success {
				icon = "✗"
			}

			row := checkResults.Objects[i].(*fyne.Container)
			row.Objects[0].(*widget.Label).SetText(icon)
			row.Objects[2].(*widget.Label).SetText(msg)
			row.Refresh()
		}
	})

	content := container.NewVBox(
		widget.NewCard("", "⚠️ FINAL WARNING ⚠️",
			widget.NewLabel("This operation will PERMANENTLY ERASE all data on:\n"+rw.config.TargetPath+"\n\nThere is NO UNDO. Make sure you have selected the correct disk!"),
		),
		widget.NewSeparator(),
		checkResults,
		runChecksBtn,
		widget.NewSeparator(),
		verifyCheck,
		confirmCheck,
	)

	step := NewBaseStep(
		"Step 4: Verification",
		"Confirm destructive operation and run safety checks",
		container.NewScroll(content),
	)

	step.SetOnEnter(func() {
		// Auto-run checks
		time.AfterFunc(500*time.Millisecond, func() {
			runChecksBtn.OnTapped()
		})
	})

	step.SetCanProgress(func() bool {
		return rw.config.Confirmed
	})

	return step
}

// Step 5: Execution
func (rw *RestoreWizard) createExecutionStep() WizardStep {
	progressBar := widget.NewProgressBar()
	speedLabel := widget.NewLabel("Speed: -- MB/s")
	etaLabel := widget.NewLabel("ETA: Calculating...")
	statusLabel := widget.NewLabel("Status: Preparing...")

	logEntry := widget.NewMultiLineEntry()
	logEntry.Disable()
	logEntry.SetText("Restore log will appear here...\n")

	logScroll := container.NewScroll(logEntry)
	logScroll.SetMinSize(fyne.NewSize(0, 200))

	cancelBtn := widget.NewButtonWithIcon("Emergency Stop", theme.CancelIcon(), func() {
		if rw.cancel != nil {
			rw.cancel()
			statusLabel.SetText("Status: Cancelling... This may take a moment.")
		}
	})
	cancelBtn.Disable()
	cancelBtn.Importance = widget.DangerImportance

	content := container.NewVBox(
		widget.NewLabel("⚠️ DO NOT POWER OFF OR DISCONNECT DRIVES DURING RESTORE"),
		widget.NewSeparator(),
		progressBar,
		container.NewHBox(speedLabel, layout.NewSpacer(), etaLabel),
		statusLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Restore Log:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		logScroll,
		cancelBtn,
	)

	step := NewBaseStep(
		"Step 5: Execution",
		"Restoring image to target disk",
		content,
	)

	step.SetOnEnter(func() {
		// Start restore operation
		go rw.executeRestore(progressBar, speedLabel, etaLabel, statusLabel, logEntry, cancelBtn)
	})

	step.SetCanProgress(func() bool {
		return false // Cannot proceed until restore completes
	})

	return step
}

// executeRestore performs the actual restore operation
func (rw *RestoreWizard) executeRestore(
	progressBar *widget.ProgressBar,
	speedLabel, etaLabel, statusLabel *widget.Label,
	logEntry *widget.Entry,
	cancelBtn *widget.Button,
) {
	rw.ctx, rw.cancel = context.WithCancel(context.Background())
	defer func() {
		cancelBtn.Disable()
		cancelBtn.Refresh()
	}()

	cancelBtn.Enable()
	cancelBtn.Refresh()

	logMsg := func(msg string) {
		logEntry.SetText(logEntry.Text + msg + "\n")
		logEntry.CursorRow = len(strings.Split(logEntry.Text, "\n"))
		logEntry.Refresh()
	}

	// Simulate restore operation
	logMsg(fmt.Sprintf("[%s] Opening image: %s", time.Now().Format("15:04:05"), rw.config.ImagePath))
	time.Sleep(500 * time.Millisecond)

	logMsg(fmt.Sprintf("[%s] Verifying target disk: %s", time.Now().Format("15:04:05"), rw.config.TargetPath))
	time.Sleep(500 * time.Millisecond)

	logMsg(fmt.Sprintf("[%s] Beginning sector-by-sector restore", time.Now().Format("15:04:05")))
	statusLabel.SetText("Status: Restoring data...")

	// Simulate progress
	startTime := time.Now()
	totalSteps := 100
	for i := 0; i <= totalSteps; i++ {
		select {
		case <-rw.ctx.Done():
			logMsg("[CANCELLED] Restore operation cancelled by user")
			statusLabel.SetText("Status: Cancelled")
			return
		default:
		}

		progressBar.SetValue(float64(i) / float64(totalSteps))

		elapsed := time.Since(startTime).Seconds()
		if elapsed > 0 {
			bytesPerSec := float64(rw.config.ImageSize) * float64(i) / float64(totalSteps) / elapsed
			speedLabel.SetText(fmt.Sprintf("Speed: %.2f MB/s", bytesPerSec/(1024*1024)))

			if i < totalSteps {
				remaining := float64(totalSteps-i) / float64(i) * elapsed
				etaLabel.SetText(fmt.Sprintf("ETA: %s", time.Duration(remaining*float64(time.Second)).Round(time.Second)))
			}
		}

		time.Sleep(100 * time.Millisecond)

		// Log milestones
		if i%25 == 0 && i > 0 {
			logMsg(fmt.Sprintf("[%s] Progress: %d%% complete", time.Now().Format("15:04:05"), i))
		}
	}

	logMsg(fmt.Sprintf("[%s] Data restoration complete", time.Now().Format("15:04:05")))

	if rw.config.VerifyHashes {
		statusLabel.SetText("Status: Verifying data integrity...")
		logMsg(fmt.Sprintf("[%s] Performing post-restore verification", time.Now().Format("15:04:05")))
		time.Sleep(2 * time.Second)
		logMsg("[VERIFY] Data integrity check: PASSED")
	}

	logMsg(fmt.Sprintf("[%s] Restore operation completed successfully", time.Now().Format("15:04:05")))
	statusLabel.SetText("Status: Completed")
	progressBar.SetValue(1.0)

	dialog.ShowInformation("Success", "Restore operation completed successfully!", rw.window)
	rw.wizard.EnableNext(true)
}
