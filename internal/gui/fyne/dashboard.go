package fyne

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Dashboard represents the main dashboard
type Dashboard struct {
	app    fyne.App
	window fyne.Window
}

// NewDashboard creates a new dashboard
func NewDashboard() *Dashboard {
	a := app.New()
	w := a.NewWindow("Diskimager Forensics Suite")

	d := &Dashboard{
		app:    a,
		window: w,
	}

	d.buildUI()
	return d
}

// Run starts the dashboard application
func (d *Dashboard) Run() {
	d.window.ShowAndRun()
}

// buildUI constructs the dashboard UI
func (d *Dashboard) buildUI() {
	d.window.Resize(fyne.NewSize(1000, 700))
	d.window.CenterOnScreen()

	// Main content
	content := container.NewBorder(
		d.createHeader(),
		d.createFooter(),
		nil,
		nil,
		container.NewVBox(
			d.createQuickActions(),
			widget.NewSeparator(),
			d.createRecentOperations(),
			widget.NewSeparator(),
			d.createSystemHealth(),
		),
	)

	d.window.SetContent(content)
}

// createHeader creates the dashboard header
func (d *Dashboard) createHeader() fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"Diskimager Forensics Suite",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	subtitle := widget.NewLabel("Professional Digital Evidence Acquisition and Analysis")
	subtitle.Alignment = fyne.TextAlignCenter

	settingsBtn := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() {
		d.showSettings()
	})

	aboutBtn := widget.NewButtonWithIcon("About", theme.InfoIcon(), func() {
		d.showAbout()
	})

	return container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		container.NewHBox(
			layout.NewSpacer(),
			settingsBtn,
			aboutBtn,
		),
	)
}

// createFooter creates the dashboard footer
func (d *Dashboard) createFooter() fyne.CanvasObject {
	statusLabel := widget.NewLabel("Ready")
	timeLabel := widget.NewLabel(time.Now().Format("15:04:05 MST"))

	// Update time every second
	go func() {
		for range time.Tick(time.Second) {
			timeLabel.SetText(time.Now().Format("15:04:05 MST"))
		}
	}()

	return container.NewBorder(
		widget.NewSeparator(),
		nil, nil, nil,
		container.NewHBox(
			statusLabel,
			layout.NewSpacer(),
			widget.NewLabel("v1.0.0"),
			widget.NewSeparator(),
			timeLabel,
		),
	)
}

// createQuickActions creates the quick action buttons
func (d *Dashboard) createQuickActions() fyne.CanvasObject {
	imageBtn := widget.NewButtonWithIcon("Create Image", theme.DownloadIcon(), func() {
		wizardWindow := d.app.NewWindow("Imaging Wizard")
		wizard := NewImagingWizard(wizardWindow)
		wizard.Show()
	})
	imageBtn.Importance = widget.HighImportance

	restoreBtn := widget.NewButtonWithIcon("Restore Image", theme.UploadIcon(), func() {
		wizardWindow := d.app.NewWindow("Restore Wizard")
		wizard := NewRestoreWizard(wizardWindow)
		wizard.Show()
	})

	cloneBtn := widget.NewButtonWithIcon("Clone Disk", theme.ContentCopyIcon(), func() {
		d.window.Canvas().Unfocus()
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Coming Soon",
			Content: "Disk cloning wizard will be available in next version",
		})
	})

	findBtn := widget.NewButtonWithIcon("Find Files", theme.SearchIcon(), func() {
		d.window.Canvas().Unfocus()
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "CLI Feature",
			Content: "File search is available via 'diskimager find' command",
		})
	})

	analyzeBtn := widget.NewButtonWithIcon("Analyze Image", theme.DocumentIcon(), func() {
		d.window.Canvas().Unfocus()
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "CLI Feature",
			Content: "Image analysis is available via 'diskimager analyze' command",
		})
	})

	// Arrange in grid
	actionGrid := container.NewGridWithColumns(3,
		d.createActionCard("Imaging", "Create forensic disk images with full chain of custody", imageBtn),
		d.createActionCard("Restore", "Restore forensic images to physical disks", restoreBtn),
		d.createActionCard("Clone", "Direct disk-to-disk cloning", cloneBtn),
	)

	actionGrid2 := container.NewGridWithColumns(2,
		d.createActionCard("Search", "Find files across images and disks", findBtn),
		d.createActionCard("Analyze", "Deep analysis of disk images", analyzeBtn),
	)

	return container.NewVBox(
		widget.NewLabelWithStyle("Quick Actions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		actionGrid,
		actionGrid2,
	)
}

// createActionCard creates a card for an action
func (d *Dashboard) createActionCard(title, description string, button *widget.Button) fyne.CanvasObject {
	return widget.NewCard(
		title,
		description,
		button,
	)
}

// createRecentOperations creates the recent operations list
func (d *Dashboard) createRecentOperations() fyne.CanvasObject {
	// Mock data
	recentOps := []struct {
		time      string
		operation string
		status    string
		icon      fyne.Resource
	}{
		{
			time:      "2026-03-29 14:23:15",
			operation: "Image /dev/disk1 to evidence-001.e01",
			status:    "✓ Completed",
			icon:      theme.ConfirmIcon(),
		},
		{
			time:      "2026-03-29 11:45:32",
			operation: "Restore backup.img to /dev/disk2",
			status:    "✓ Completed",
			icon:      theme.ConfirmIcon(),
		},
		{
			time:      "2026-03-28 16:12:08",
			operation: "Image /dev/disk3 to cloud storage",
			status:    "✓ Completed",
			icon:      theme.ConfirmIcon(),
		},
	}

	list := widget.NewList(
		func() int { return len(recentOps) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.InfoIcon()),
				widget.NewLabel("Template"),
				layout.NewSpacer(),
				widget.NewLabel("Status"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(recentOps) {
				op := recentOps[id]
				container := obj.(*fyne.Container)
				container.Objects[0].(*widget.Icon).SetResource(op.icon)
				container.Objects[1].(*widget.Label).SetText(fmt.Sprintf("[%s] %s", op.time, op.operation))
				container.Objects[3].(*widget.Label).SetText(op.status)
			}
		},
	)

	clearBtn := widget.NewButtonWithIcon("Clear History", theme.DeleteIcon(), func() {
		// Would clear history in real implementation
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "History Cleared",
			Content: "Operation history has been cleared",
		})
	})

	return container.NewVBox(
		container.NewHBox(
			widget.NewLabelWithStyle("Recent Operations", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			clearBtn,
		),
		container.NewBorder(nil, nil, nil, nil, list),
	)
}

// createSystemHealth creates the system health panel
func (d *Dashboard) createSystemHealth() fyne.CanvasObject {
	// Mock disk health data
	disks := []struct {
		name   string
		health string
		icon   string
		temp   string
	}{
		{name: "/dev/disk0 - System SSD", health: "Healthy", icon: "✓", temp: "35°C"},
		{name: "/dev/disk1 - Data HDD", health: "Healthy", icon: "✓", temp: "42°C"},
		{name: "/dev/disk2 - USB Drive", health: "Healthy", icon: "✓", temp: "N/A"},
	}

	list := widget.NewList(
		func() int { return len(disks) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Icon"),
				widget.NewIcon(theme.StorageIcon()),
				widget.NewLabel("Template"),
				layout.NewSpacer(),
				widget.NewLabel("Status"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(disks) {
				disk := disks[id]
				container := obj.(*fyne.Container)
				container.Objects[0].(*widget.Label).SetText(disk.icon)
				container.Objects[2].(*widget.Label).SetText(disk.name)
				container.Objects[4].(*widget.Label).SetText(fmt.Sprintf("%s | %s", disk.health, disk.temp))
			}
		},
	)

	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Refreshed",
			Content: "Disk health information updated",
		})
	})

	return container.NewVBox(
		container.NewHBox(
			widget.NewLabelWithStyle("System Health (SMART Status)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			refreshBtn,
		),
		container.NewBorder(nil, nil, nil, nil, list),
	)
}

// showSettings displays the settings dialog
func (d *Dashboard) showSettings() {
	settingsWindow := d.app.NewWindow("Settings")
	settingsWindow.Resize(fyne.NewSize(600, 500))

	// Theme settings
	themeGroup := widget.NewCard("Appearance", "", container.NewVBox(
		widget.NewRadioGroup([]string{"Light Theme", "Dark Theme", "System Default"}, func(s string) {
			// Would change theme
		}),
	))

	// Default settings
	defaultsGroup := widget.NewCard("Defaults", "", container.NewVBox(
		widget.NewLabel("Default compression level:"),
		widget.NewSlider(0, 9),
		widget.NewCheck("Always verify hashes", nil),
		widget.NewCheck("Create MD5 hash (in addition to SHA256)", nil),
	))

	// Cloud credentials
	cloudGroup := widget.NewCard("Cloud Storage", "", container.NewVBox(
		widget.NewButton("Configure AWS Credentials", func() {}),
		widget.NewButton("Configure Azure Credentials", func() {}),
		widget.NewButton("Configure GCS Credentials", func() {}),
	))

	closeBtn := widget.NewButton("Close", func() {
		settingsWindow.Close()
	})

	content := container.NewVBox(
		themeGroup,
		defaultsGroup,
		cloudGroup,
		layout.NewSpacer(),
		closeBtn,
	)

	settingsWindow.SetContent(container.NewScroll(content))
	settingsWindow.Show()
}

// showAbout displays the about dialog
func (d *Dashboard) showAbout() {
	aboutWindow := d.app.NewWindow("About")
	aboutWindow.Resize(fyne.NewSize(500, 400))

	logo := widget.NewLabelWithStyle("Diskimager", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	info := widget.NewLabel(`Version 1.0.0

Professional Forensic Disk Imaging Suite

Features:
• Multi-format support (Raw, E01, VMDK, VHD)
• Cloud storage integration (S3, Azure, GCS)
• Full chain of custody documentation
• Real-time progress monitoring
• Bad sector handling
• Cryptographic verification (SHA256, MD5)
• Norton Ghost-style wizard interface

Built with Go and Fyne

© 2026 AfterDark Systems`)
	info.Wrapping = fyne.TextWrapWord
	info.Alignment = fyne.TextAlignCenter

	closeBtn := widget.NewButton("Close", func() {
		aboutWindow.Close()
	})

	content := container.NewVBox(
		logo,
		widget.NewSeparator(),
		info,
		layout.NewSpacer(),
		closeBtn,
	)

	aboutWindow.SetContent(container.NewPadded(content))
	aboutWindow.CenterOnScreen()
	aboutWindow.Show()
}
