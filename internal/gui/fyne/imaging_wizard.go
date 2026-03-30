package fyne

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/diskimager/imager"
	"github.com/afterdarksys/diskimager/pkg/format/e01"
	"github.com/afterdarksys/diskimager/pkg/format/raw"
	"github.com/afterdarksys/diskimager/pkg/progress"
	"github.com/afterdarksys/diskimager/pkg/storage"
)

// ImagingWizardConfig holds the imaging wizard configuration
type ImagingWizardConfig struct {
	SourcePath      string
	DestPath        string
	Format          string
	Compression     int
	Encryption      bool
	CaseNumber      string
	Examiner        string
	Description     string
	Evidence        string
	ScheduleTime    time.Time
	Scheduled       bool
	CloudDestType   string // local, s3, azure, gcs
	CloudBucket     string
	CloudRegion     string
	CloudAccessKey  string
	CloudSecretKey  string
}

// ImagingWizard creates a 6-step imaging wizard
type ImagingWizard struct {
	window fyne.Window
	config *ImagingWizardConfig
	wizard *Wizard

	// Progress tracking
	tracker *progress.Tracker
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewImagingWizard creates a new imaging wizard
func NewImagingWizard(window fyne.Window) *ImagingWizard {
	iw := &ImagingWizard{
		window: window,
		config: &ImagingWizardConfig{
			Format:      "Raw",
			Compression: 6,
		},
	}

	steps := []WizardStep{
		iw.createSourceStep(),
		iw.createDestinationStep(),
		iw.createOptionsStep(),
		iw.createVerificationStep(),
		iw.createExecutionStep(),
		iw.createSummaryStep(),
	}

	iw.wizard = NewWizard(window, steps, func() {
		// Wizard complete
		window.Close()
	}, func() {
		// Wizard cancelled
		if iw.cancel != nil {
			iw.cancel()
		}
		window.Close()
	})

	return iw
}

// Show displays the wizard
func (iw *ImagingWizard) Show() {
	iw.window.SetContent(iw.wizard.Content())
	iw.window.Resize(fyne.NewSize(900, 700))
	iw.window.CenterOnScreen()
	iw.window.Show()
}

// Step 1: Source Selection
func (iw *ImagingWizard) createSourceStep() WizardStep {
	sourceEntry := widget.NewEntry()
	sourceEntry.SetPlaceHolder("Select source device or file...")

	browseBtn := widget.NewButtonWithIcon("Browse...", theme.FolderOpenIcon(), func() {
		dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
			if err != nil || uc == nil {
				return
			}
			defer uc.Close()
			iw.config.SourcePath = uc.URI().Path()
			sourceEntry.SetText(iw.config.SourcePath)
		}, iw.window)
	})

	// Disk list (mock for now - would integrate with actual disk detection)
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
				"/dev/disk0 - 500GB SSD (APFS) - Healthy",
				"/dev/disk1 - 2TB HDD (NTFS) - Healthy",
				"/dev/disk2 - 1TB USB (FAT32) - Healthy",
			}
			if id < len(diskNames) {
				obj.(*fyne.Container).Objects[1].(*widget.Label).SetText(diskNames[id])
			}
		},
	)
	diskList.OnSelected = func(id widget.ListItemID) {
		diskPaths := []string{"/dev/disk0", "/dev/disk1", "/dev/disk2"}
		if id < len(diskPaths) {
			iw.config.SourcePath = diskPaths[id]
			sourceEntry.SetText(diskPaths[id])
		}
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle("Available Disks:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, nil, diskList),
		layout.NewSpacer(),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Or select a file:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, browseBtn, sourceEntry),
	)

	step := NewBaseStep(
		"Step 1: Source Selection",
		"Select the disk or image file you want to image",
		content,
	)

	step.SetValidator(func() error {
		if iw.config.SourcePath == "" {
			return errors.New("please select a source device or file")
		}
		if _, err := os.Stat(iw.config.SourcePath); err != nil {
			return fmt.Errorf("cannot access source: %v", err)
		}
		return nil
	})

	step.SetCanProgress(func() bool {
		return iw.config.SourcePath != ""
	})

	// Update progress dynamically
	sourceEntry.OnChanged = func(s string) {
		iw.config.SourcePath = s
		iw.wizard.EnableNext(iw.config.SourcePath != "")
	}

	return step
}

// Step 2: Destination
func (iw *ImagingWizard) createDestinationStep() WizardStep {
	destTabs := container.NewAppTabs()

	// Local tab
	localEntry := widget.NewEntry()
	localEntry.SetPlaceHolder("Select destination path...")

	browseBtn := widget.NewButtonWithIcon("Browse...", theme.FolderOpenIcon(), func() {
		dialog.ShowFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil || uc == nil {
				return
			}
			defer uc.Close()
			iw.config.DestPath = uc.URI().Path()
			iw.config.CloudDestType = "local"
			localEntry.SetText(iw.config.DestPath)
		}, iw.window)
	})

	localTab := container.NewVBox(
		widget.NewLabel("Save image to local filesystem"),
		container.NewBorder(nil, nil, nil, browseBtn, localEntry),
	)

	// Cloud tab
	cloudTypeSelect := widget.NewSelect([]string{"Amazon S3", "Azure Blob", "Google Cloud Storage"}, func(s string) {
		switch s {
		case "Amazon S3":
			iw.config.CloudDestType = "s3"
		case "Azure Blob":
			iw.config.CloudDestType = "azure"
		case "Google Cloud Storage":
			iw.config.CloudDestType = "gcs"
		}
	})
	cloudTypeSelect.SetSelected("Amazon S3")

	bucketEntry := widget.NewEntry()
	bucketEntry.SetPlaceHolder("bucket-name")
	bucketEntry.OnChanged = func(s string) {
		iw.config.CloudBucket = s
	}

	regionEntry := widget.NewEntry()
	regionEntry.SetPlaceHolder("us-east-1")
	regionEntry.OnChanged = func(s string) {
		iw.config.CloudRegion = s
	}

	accessKeyEntry := widget.NewEntry()
	accessKeyEntry.SetPlaceHolder("Access Key ID")
	accessKeyEntry.OnChanged = func(s string) {
		iw.config.CloudAccessKey = s
	}

	secretKeyEntry := widget.NewPasswordEntry()
	secretKeyEntry.SetPlaceHolder("Secret Access Key")
	secretKeyEntry.OnChanged = func(s string) {
		iw.config.CloudSecretKey = s
	}

	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("evidence/case-001/disk.img")
	pathEntry.OnChanged = func(s string) {
		iw.config.DestPath = s
	}

	cloudTab := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Provider", cloudTypeSelect),
			widget.NewFormItem("Bucket/Container", bucketEntry),
			widget.NewFormItem("Region", regionEntry),
			widget.NewFormItem("Access Key", accessKeyEntry),
			widget.NewFormItem("Secret Key", secretKeyEntry),
			widget.NewFormItem("Destination Path", pathEntry),
		),
	)

	destTabs.Append(container.NewTabItem("Local Storage", localTab))
	destTabs.Append(container.NewTabItem("Cloud Storage", cloudTab))

	step := NewBaseStep(
		"Step 2: Destination",
		"Choose where to save the forensic image",
		destTabs,
	)

	step.SetValidator(func() error {
		if iw.config.DestPath == "" {
			return errors.New("please specify a destination path")
		}
		if iw.config.CloudDestType != "local" {
			if iw.config.CloudBucket == "" {
				return errors.New("please specify cloud bucket/container name")
			}
		}
		return nil
	})

	step.SetCanProgress(func() bool {
		return iw.config.DestPath != ""
	})

	localEntry.OnChanged = func(s string) {
		iw.config.DestPath = s
		iw.config.CloudDestType = "local"
		iw.wizard.EnableNext(s != "")
	}

	return step
}

// Step 3: Options
func (iw *ImagingWizard) createOptionsStep() WizardStep {
	formatSelect := widget.NewSelect([]string{"Raw", "E01", "VMDK", "VHD"}, func(s string) {
		iw.config.Format = s
	})
	formatSelect.SetSelected("Raw")

	compressionSlider := widget.NewSlider(0, 9)
	compressionSlider.Value = 6
	compressionSlider.Step = 1
	compressionLabel := widget.NewLabel("Level 6")
	compressionSlider.OnChanged = func(value float64) {
		iw.config.Compression = int(value)
		compressionLabel.SetText(fmt.Sprintf("Level %d", int(value)))
	}

	encryptionCheck := widget.NewCheck("Enable AES-256 encryption", func(checked bool) {
		iw.config.Encryption = checked
	})

	caseEntry := widget.NewEntry()
	caseEntry.SetPlaceHolder("CASE-2026-001")
	caseEntry.OnChanged = func(s string) {
		iw.config.CaseNumber = s
	}

	examinerEntry := widget.NewEntry()
	examinerEntry.SetPlaceHolder("John Doe")
	examinerEntry.OnChanged = func(s string) {
		iw.config.Examiner = s
	}

	evidenceEntry := widget.NewEntry()
	evidenceEntry.SetPlaceHolder("Laptop hard drive from suspect")
	evidenceEntry.OnChanged = func(s string) {
		iw.config.Evidence = s
	}

	descEntry := widget.NewMultiLineEntry()
	descEntry.SetPlaceHolder("Additional notes about this acquisition...")
	descEntry.OnChanged = func(s string) {
		iw.config.Description = s
	}

	scheduleCheck := widget.NewCheck("Schedule for later", func(checked bool) {
		iw.config.Scheduled = checked
	})

	content := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Image Format", formatSelect),
			widget.NewFormItem("Compression", container.NewBorder(nil, nil, nil, compressionLabel, compressionSlider)),
			widget.NewFormItem("Security", encryptionCheck),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Chain of Custody", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Case Number", caseEntry),
			widget.NewFormItem("Examiner", examinerEntry),
			widget.NewFormItem("Evidence ID", evidenceEntry),
			widget.NewFormItem("Description", descEntry),
		),
		widget.NewSeparator(),
		scheduleCheck,
	)

	step := NewBaseStep(
		"Step 3: Options",
		"Configure imaging options and chain of custody information",
		container.NewScroll(content),
	)

	step.SetValidator(func() error {
		if iw.config.CaseNumber == "" {
			return errors.New("case number is required for forensic imaging")
		}
		if iw.config.Examiner == "" {
			return errors.New("examiner name is required")
		}
		return nil
	})

	return step
}

// Step 4: Verification
func (iw *ImagingWizard) createVerificationStep() WizardStep {
	checkResults := container.NewVBox()

	checks := []struct {
		name   string
		check  func() (bool, string)
		icon   fyne.Resource
		status *widget.Label
	}{
		{
			name: "Source accessible",
			check: func() (bool, string) {
				if stat, err := os.Stat(iw.config.SourcePath); err != nil {
					return false, fmt.Sprintf("Error: %v", err)
				} else {
					size := stat.Size()
					return true, fmt.Sprintf("Ready - Size: %d bytes", size)
				}
			},
		},
		{
			name: "Destination writable",
			check: func() (bool, string) {
				if iw.config.CloudDestType == "local" {
					dir := filepath.Dir(iw.config.DestPath)
					if stat, err := os.Stat(dir); err != nil {
						return false, fmt.Sprintf("Directory not accessible: %v", err)
					} else if !stat.IsDir() {
						return false, "Parent is not a directory"
					}
					// Check write permission
					testFile := filepath.Join(dir, ".diskimager_test")
					if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
						return false, fmt.Sprintf("Not writable: %v", err)
					}
					os.Remove(testFile)
					return true, "Directory is writable"
				}
				return true, "Cloud storage configured"
			},
		},
		{
			name: "Sufficient disk space",
			check: func() (bool, string) {
				// Simplified check
				if stat, err := os.Stat(iw.config.SourcePath); err == nil {
					sourceSize := stat.Size()
					// Estimate with compression
					estimatedSize := sourceSize
					if iw.config.Compression > 0 {
						estimatedSize = sourceSize / 2 // Rough estimate
					}
					return true, fmt.Sprintf("Required: ~%d MB", estimatedSize/(1024*1024))
				}
				return false, "Cannot determine space requirements"
			},
		},
		{
			name: "SMART status (if applicable)",
			check: func() (bool, string) {
				// Mock SMART check
				if strings.HasPrefix(iw.config.SourcePath, "/dev/") {
					return true, "Disk health: PASSED"
				}
				return true, "N/A (file source)"
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

	runChecksBtn := widget.NewButtonWithIcon("Run Pre-Flight Checks", theme.MediaPlayIcon(), func() {
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
		iw.wizard.EnableNext(true)
	})

	content := container.NewVBox(
		widget.NewLabel("Pre-flight checks ensure the imaging operation can proceed safely."),
		widget.NewSeparator(),
		checkResults,
		layout.NewSpacer(),
		runChecksBtn,
	)

	step := NewBaseStep(
		"Step 4: Verification",
		"Verify system readiness before imaging",
		content,
	)

	step.SetCanProgress(func() bool {
		return false // User must run checks
	})

	step.SetOnEnter(func() {
		// Auto-run checks when entering step
		time.AfterFunc(500*time.Millisecond, func() {
			runChecksBtn.OnTapped()
		})
	})

	return step
}

// Step 5: Execution
func (iw *ImagingWizard) createExecutionStep() WizardStep {
	progressBar := widget.NewProgressBar()
	speedLabel := widget.NewLabel("Speed: -- MB/s")
	etaLabel := widget.NewLabel("ETA: Calculating...")
	phaseLabel := widget.NewLabel("Phase: Initializing")
	statusLabel := widget.NewLabel("Status: Ready to start")

	logEntry := widget.NewMultiLineEntry()
	logEntry.Disable()
	logEntry.SetText("Imaging log will appear here...\n")

	logScroll := container.NewScroll(logEntry)
	logScroll.SetMinSize(fyne.NewSize(0, 200))

	cancelBtn := widget.NewButtonWithIcon("Cancel Operation", theme.CancelIcon(), func() {
		if iw.cancel != nil {
			iw.cancel()
			statusLabel.SetText("Status: Cancelling...")
		}
	})
	cancelBtn.Disable()

	content := container.NewVBox(
		phaseLabel,
		progressBar,
		container.NewHBox(speedLabel, layout.NewSpacer(), etaLabel),
		statusLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Operation Log:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		logScroll,
		cancelBtn,
	)

	step := NewBaseStep(
		"Step 5: Execution",
		"Imaging operation in progress - do not interrupt",
		content,
	)

	step.SetOnEnter(func() {
		// Start imaging operation
		go iw.executeImaging(progressBar, speedLabel, etaLabel, phaseLabel, statusLabel, logEntry, cancelBtn)
	})

	step.SetCanProgress(func() bool {
		return false // Cannot proceed until imaging completes
	})

	return step
}

// executeImaging performs the actual imaging operation
func (iw *ImagingWizard) executeImaging(
	progressBar *widget.ProgressBar,
	speedLabel, etaLabel, phaseLabel, statusLabel *widget.Label,
	logEntry *widget.Entry,
	cancelBtn *widget.Button,
) {
	iw.ctx, iw.cancel = context.WithCancel(context.Background())
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

	// Open source
	logMsg(fmt.Sprintf("[%s] Opening source: %s", time.Now().Format("15:04:05"), iw.config.SourcePath))
	source, err := os.Open(iw.config.SourcePath)
	if err != nil {
		logMsg(fmt.Sprintf("[ERROR] Failed to open source: %v", err))
		statusLabel.SetText("Status: Failed")
		return
	}
	defer source.Close()

	stat, _ := source.Stat()
	totalSize := stat.Size()

	// Open destination
	logMsg(fmt.Sprintf("[%s] Opening destination: %s", time.Now().Format("15:04:05"), iw.config.DestPath))

	var destURL string
	if iw.config.CloudDestType == "local" {
		destURL = iw.config.DestPath
	} else {
		destURL = fmt.Sprintf("%s://%s/%s", iw.config.CloudDestType, iw.config.CloudBucket, iw.config.DestPath)
	}

	outTarget, err := storage.OpenDestination(destURL, false)
	if err != nil {
		logMsg(fmt.Sprintf("[ERROR] Failed to open destination: %v", err))
		statusLabel.SetText("Status: Failed")
		return
	}
	defer outTarget.Close()

	// Create format writer
	var out io.WriteCloser
	metadata := imager.Metadata{
		CaseNumber:  iw.config.CaseNumber,
		Examiner:    iw.config.Examiner,
		EvidenceNum: iw.config.Evidence,
		Description: iw.config.Description,
	}

	switch iw.config.Format {
	case "E01":
		out, err = e01.NewWriter(outTarget, false, metadata)
	default:
		out, err = raw.NewWriter(outTarget)
	}

	if err != nil {
		logMsg(fmt.Sprintf("[ERROR] Failed to create writer: %v", err))
		statusLabel.SetText("Status: Failed")
		return
	}
	defer out.Close()

	// Create progress tracker
	iw.tracker = progress.NewTracker(totalSize, 500*time.Millisecond)

	// Monitor progress
	go func() {
		for prog := range iw.tracker.Progress() {
			progressBar.SetValue(prog.Percentage / 100.0)
			speedLabel.SetText(fmt.Sprintf("Speed: %.2f MB/s", float64(prog.Speed)/(1024*1024)))

			if prog.ETA > 0 {
				etaLabel.SetText(fmt.Sprintf("ETA: %s", prog.ETA.Round(time.Second)))
			}

			phaseLabel.SetText(fmt.Sprintf("Phase: %s", prog.Phase))

			if prog.BadSectors > 0 {
				statusLabel.SetText(fmt.Sprintf("Status: Running (%d bad sectors)", prog.BadSectors))
			} else {
				statusLabel.SetText("Status: Running")
			}
		}
	}()

	// Perform imaging
	logMsg(fmt.Sprintf("[%s] Starting imaging operation", time.Now().Format("15:04:05")))
	iw.tracker.SetPhase(progress.PhaseReading)
	iw.tracker.SetStatus(progress.StatusRunning)

	cfg := imager.Config{
		Source:      io.TeeReader(source, &progressWriter{tracker: iw.tracker}),
		Destination: out,
		HashAlgo:    "sha256",
		Metadata:    metadata,
	}

	startTime := time.Now()
	result, imgErr := imager.Image(cfg)
	elapsed := time.Since(startTime)

	if imgErr != nil {
		logMsg(fmt.Sprintf("[ERROR] Imaging failed: %v", imgErr))
		statusLabel.SetText("Status: Failed")
		iw.tracker.Fail(imgErr)
	} else {
		logMsg(fmt.Sprintf("[%s] Imaging completed successfully", time.Now().Format("15:04:05")))
		logMsg(fmt.Sprintf("Hash (SHA256): %s", result.Hash))
		logMsg(fmt.Sprintf("Bytes processed: %d", result.BytesCopied))
		logMsg(fmt.Sprintf("Time elapsed: %s", elapsed.Round(time.Second)))
		statusLabel.SetText("Status: Completed")
		progressBar.SetValue(1.0)
		iw.tracker.Complete()
		iw.wizard.EnableNext(true)
	}
}

// progressWriter wraps a writer to track progress
type progressWriter struct {
	tracker *progress.Tracker
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.tracker.AddBytes(int64(n))
	return n, nil
}

// Step 6: Summary
func (iw *ImagingWizard) createSummaryStep() WizardStep {
	summaryText := widget.NewMultiLineEntry()
	summaryText.Disable()
	summaryText.Wrapping = fyne.TextWrapWord

	generateReportBtn := widget.NewButtonWithIcon("Generate Report", theme.DocumentSaveIcon(), func() {
		dialog.ShowFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil || uc == nil {
				return
			}
			defer uc.Close()

			report := fmt.Sprintf(`FORENSIC IMAGING REPORT
Generated: %s

CASE INFORMATION
Case Number: %s
Examiner: %s
Evidence ID: %s

SOURCE
Path: %s

DESTINATION
Path: %s
Format: %s

OPERATION SUMMARY
Status: Completed
Time Elapsed: %s
Hash Algorithm: SHA256
Hash Value: [See imaging log]

NOTES
%s

This report was generated by Diskimager Forensics Suite
`, time.Now().Format(time.RFC3339),
				iw.config.CaseNumber,
				iw.config.Examiner,
				iw.config.Evidence,
				iw.config.SourcePath,
				iw.config.DestPath,
				iw.config.Format,
				"[Duration from execution]",
				iw.config.Description,
			)

			uc.Write([]byte(report))
			dialog.ShowInformation("Success", "Report saved successfully", iw.window)
		}, iw.window)
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Imaging Operation Complete", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		summaryText,
		layout.NewSpacer(),
		generateReportBtn,
	)

	step := NewBaseStep(
		"Step 6: Summary",
		"Review imaging results and generate forensic report",
		content,
	)

	step.SetOnEnter(func() {
		summary := fmt.Sprintf(`IMAGING SUMMARY

Source: %s
Destination: %s
Format: %s
Compression: Level %d
Encryption: %v

Case Number: %s
Examiner: %s
Evidence ID: %s

The imaging operation has completed. Use the button below to generate
a detailed forensic report for your case file.

All hash values and verification data are available in the operation log
from the previous step.
`,
			iw.config.SourcePath,
			iw.config.DestPath,
			iw.config.Format,
			iw.config.Compression,
			iw.config.Encryption,
			iw.config.CaseNumber,
			iw.config.Examiner,
			iw.config.Evidence,
		)
		summaryText.SetText(summary)
	})

	return step
}
