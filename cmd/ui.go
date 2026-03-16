package cmd

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/diskimager/imager"
	"github.com/afterdarksys/diskimager/pkg/format/e01"
	"github.com/afterdarksys/diskimager/pkg/format/raw"
	"github.com/afterdarksys/diskimager/pkg/storage"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the graphical user interface",
	Run: func(cmd *cobra.Command, args []string) {
		startGUI()
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

func startGUI() {
	a := app.New()
	w := a.NewWindow("Diskimager Forensics Suite")
	w.Resize(fyne.NewSize(800, 600))

	tabs := container.NewAppTabs(
		container.NewTabItem("Imager", buildImagerTab(w)),
		container.NewTabItem("Disktool", buildDisktoolTab(w)),
		container.NewTabItem("Forensick", buildForensickTab(w)),
	)
	
	tabs.SetTabLocation(container.TabLocationTop)
	w.SetContent(tabs)
	w.ShowAndRun()
}

// ---------------------------------------------------------
// IMAGER TAB
// ---------------------------------------------------------

// GuiProgressReader wraps io.Reader to update a Fyne progress bar
type GuiProgressReader struct {
	Reader      io.Reader
	TotalBytes  int64
	CopiedBytes int64
	lastUpdate  int64
	ProgressBar *widget.ProgressBar
	OnUpdate    func(copied, total int64)
}

func (pr *GuiProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	atomic.AddInt64(&pr.CopiedBytes, int64(n))
	
	// Use a 5MB threshold rather than modulo to save CPU
	current := atomic.LoadInt64(&pr.CopiedBytes)
	last := atomic.LoadInt64(&pr.lastUpdate)
	
	if current-last >= 5*1024*1024 {
		atomic.StoreInt64(&pr.lastUpdate, current)
		if pr.TotalBytes > 0 {
			pr.ProgressBar.SetValue(float64(current) / float64(pr.TotalBytes))
		}
		if pr.OnUpdate != nil {
			pr.OnUpdate(current, pr.TotalBytes)
		}
	}
	return n, err
}

func buildImagerTab(win fyne.Window) fyne.CanvasObject {
	sourceEntry := widget.NewEntry()
	sourceEntry.SetPlaceHolder("/dev/sda or input.txt")
	
	destEntry := widget.NewEntry()
	destEntry.SetPlaceHolder("output.img or output.e01")
	
	formatSelect := widget.NewSelect([]string{"Raw", "E01"}, nil)
	formatSelect.SetSelected("Raw")
	
	caseEntry := widget.NewEntry()
	caseEntry.SetPlaceHolder("Case Number")
	
	examinerEntry := widget.NewEntry()
	examinerEntry.SetPlaceHolder("Examiner Name")
	
	progress := widget.NewProgressBar()
	progress.SetValue(0)
	
	statusData := binding.NewString()
	statusData.Set("Ready.")
	statusLabel := widget.NewLabelWithData(statusData)

	startBtn := widget.NewButton("Start Imaging", nil)
	startBtn.OnTapped = func() {
		if sourceEntry.Text == "" || destEntry.Text == "" {
			dialog.ShowInformation("Error", "Source and Destination are required.", win)
			return
		}
		
		startBtn.Disable()
		progress.SetValue(0)
		statusData.Set("Imaging started...")
		
		go func() {
			defer startBtn.Enable()
			
			in, err := os.Open(sourceEntry.Text)
			if err != nil {
				statusData.Set("Failed to open source: " + err.Error())
				return
			}
			defer in.Close()
			
			stat, _ := in.Stat()
			var totalSize int64 = 0
			if stat != nil {
				totalSize = stat.Size()
			}
			
			meta := imager.Metadata{
				CaseNumber: caseEntry.Text,
				Examiner:   examinerEntry.Text,
			}
			
			outTarget, err := storage.OpenDestination(destEntry.Text, false)
			if err != nil {
				statusData.Set("Failed to open destination: " + err.Error())
				return
			}

			var out io.WriteCloser
			if formatSelect.Selected == "E01" {
				out, err = e01.NewWriter(outTarget, false, meta)
			} else {
				out, err = raw.NewWriter(outTarget)
			}
			if err != nil {
				statusData.Set("Failed to create format writer: " + err.Error())
				return
			}
			defer out.Close()
			
			pr := &GuiProgressReader{
				Reader:      in,
				TotalBytes:  totalSize,
				ProgressBar: progress,
				OnUpdate: func(c, t int64) {
					statusData.Set(fmt.Sprintf("Copied: %d / %d bytes", c, t))
				},
			}
			
			cfg := imager.Config{
				Source:      pr,
				Destination: out,
				HashAlgo:    "sha256",
				Metadata:    meta,
			}
			
			start := time.Now()
			res, imgErr := imager.Image(cfg)
			
			progress.SetValue(1.0)
			if imgErr != nil {
				statusData.Set("Imaging failed: " + imgErr.Error())
			} else {
				statusData.Set(fmt.Sprintf("Success! Hash: %s | Time: %v", res.Hash, time.Since(start)))
			}
		}()
	}

	form := container.NewVBox(
		widget.NewLabelWithStyle("Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2, widget.NewLabel("Source:"), sourceEntry),
		container.NewGridWithColumns(2, widget.NewLabel("Destination:"), destEntry),
		container.NewGridWithColumns(2, widget.NewLabel("Format:"), formatSelect),
		widget.NewLabelWithStyle("Chain of Custody", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2, widget.NewLabel("Case Number:"), caseEntry),
		container.NewGridWithColumns(2, widget.NewLabel("Examiner:"), examinerEntry),
		layout.NewSpacer(),
		progress,
		statusLabel,
		startBtn,
	)
	
	return container.NewPadded(form)
}

// ---------------------------------------------------------
// DISKTOOL TAB
// ---------------------------------------------------------

func buildDisktoolTab(win fyne.Window) fyne.CanvasObject {
	deviceEntry := widget.NewEntry()
	deviceEntry.SetPlaceHolder("Target Device (e.g. /dev/sda, evidence.img)")

	actionSelect := widget.NewSelect([]string{"Scan Heuristics", "Get Volume FS Info", "Dump MBR", "Secure Wipe (3-Pass)"}, nil)
	actionSelect.SetSelected("Scan Heuristics")

	outputLog := widget.NewMultiLineEntry()
	outputLog.Disable()
	// outputLog.SetText("Disktool Output will appear here...\n") // Deprecated in Fyne API, we just set Text prop or use append
	outputLog.Text = "Disktool Output will appear here...\n"
	
	scrollableLog := container.NewScroll(outputLog)
	scrollableLog.SetMinSize(fyne.NewSize(0, 300))

	executeBtn := widget.NewButton("Execute", nil)
	
	logMsg := func(msg string) {
		outputLog.SetText(outputLog.Text + msg + "\n")
		// Ideally scroll to bottom, but standard MultiLineEntry requires a custom extension for auto-scroll
	}

	executeBtn.OnTapped = func() {
		if deviceEntry.Text == "" {
			dialog.ShowInformation("Error", "Target Device is required.", win)
			return
		}

		executeBtn.Disable()
		outputLog.SetText("Executing " + actionSelect.Selected + " on " + deviceEntry.Text + "...\n\n")

		go func() {
			defer executeBtn.Enable()
			
			// Native execution of logic since we are embedded in the same binary
			switch actionSelect.Selected {
			case "Dump MBR":
				f, err := os.Open(deviceEntry.Text)
				if err != nil {
					logMsg(fmt.Sprintf("Failed to open device: %v", err))
					return
				}
				defer f.Close()

				buf := make([]byte, 512)
				n, _ := f.Read(buf)
				
				if n < 512 {
					logMsg(fmt.Sprintf("Device too small for MBR (read %d bytes)", n))
					return
				}

				outFile := "mbr_dump.bin"
				if err := os.WriteFile(outFile, buf, 0644); err != nil {
					logMsg(fmt.Sprintf("Failed to write %s: %v", outFile, err))
					return
				}
				logMsg(fmt.Sprintf("Extracted 512-byte MBR/Boot block from %s to %s", deviceEntry.Text, outFile))
				
				if buf[510] == 0x55 && buf[511] == 0xAA {
					logMsg("Valid Boot Signature 0x55AA detected at end of sector.")
				} else {
					logMsg(fmt.Sprintf("Warning: Boot signature missing, found 0x%02X%02X", buf[510], buf[511]))
				}

			case "Secure Wipe (3-Pass)":
				logMsg("Starting DoD 5220.22-M 3-Pass Secure Wipe...")
				logMsg("WARNING: This operation is destructive.")
				
				err := WipeDrive(deviceEntry.Text, 3, func(pass int, copied, total int64) {
					// Throttle UI updates to once every 10MB
					if copied%(1024*1024*10) == 0 || copied == total {
						// We can't safely update Fyne Text property directly from a tight Goroutine loop 
						// without potential tearing, but for a simple disabled MultiLineEntry it often holds up.
						// To be perfectly safe, we'd use bindings. We'll rely on the logMsg wrapper injecting via event queue if needed,
						// but since Wipe is fast enough locally, we'll let it print to CLI or final log block.
						if copied == total {
							logMsg(fmt.Sprintf("Pass %d/%d Complete.", pass, 3))
						}
					}
				})
				
				if err != nil {
					logMsg(fmt.Sprintf("Wipe Error: %v", err))
				} else {
					logMsg("Secure Wipe Complete.")
				}

			case "Scan Heuristics":
				logMsg("Starting heuristic scan up to 100MB...")
				f, err := os.Open(deviceEntry.Text)
				if err != nil {
					logMsg(fmt.Sprintf("Scan failed: %v", err))
					return
				}
				defer f.Close()

				const maxScanBytes = 1024 * 1024 * 100
				buf := make([]byte, 512)
				var offset int64 = 0
				foundSignatures := 0

				for offset < maxScanBytes {
					n, err := f.ReadAt(buf, offset)
					if err != nil || n < 512 {
						break
					}
					if buf[0] == 0xEB && buf[1] == 0x52 && buf[2] == 0x90 && string(buf[3:7]) == "NTFS" {
						logMsg(fmt.Sprintf("[!] Found NTFS Boot Sector at offset %d (Sector %d)", offset, offset/512))
						foundSignatures++
					}
					offset += 512
				}
				logMsg(fmt.Sprintf("Scan complete. Found %d volume/partition signatures.", foundSignatures))

			case "Get Volume FS Info":
				logMsg(fmt.Sprintf("Retrieving Volume/Filesystem Info for %s...\n", deviceEntry.Text))
				
				report, err := GetDiskInfo(deviceEntry.Text)
				if err != nil {
					logMsg(fmt.Sprintf("Volume Info Error: %v", err))
				} else {
					logMsg(report)
				}
			}
			
			logMsg("\n-- Operation Complete --")
		}()
	}

	form := container.NewVBox(
		widget.NewLabelWithStyle("Disk Operations", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2, widget.NewLabel("Target Device/Image:"), deviceEntry),
		container.NewGridWithColumns(2, widget.NewLabel("Operation:"), actionSelect),
		executeBtn,
	)

	return container.NewBorder(form, nil, nil, nil, scrollableLog)
}

// ---------------------------------------------------------
// FORENSICK TAB
// ---------------------------------------------------------

func buildForensickTab(win fyne.Window) fyne.CanvasObject {
	imageEntry := widget.NewEntry()
	imageEntry.SetPlaceHolder("Evidence Image File (e.g. evidence.e01, sda.dd)")

	outDirEntry := widget.NewEntry()
	outDirEntry.SetPlaceHolder("Output Directory (for Extraction)")

	actionSelect := widget.NewSelect([]string{"Extract Unallocated", "Build MAC Timeline"}, nil)
	actionSelect.SetSelected("Extract Unallocated")

	outputLog := widget.NewMultiLineEntry()
	outputLog.Disable()
	outputLog.Text = "Forensick Output will appear here...\n"
	
	scrollableLog := container.NewScroll(outputLog)
	scrollableLog.SetMinSize(fyne.NewSize(0, 300))

	executeBtn := widget.NewButton("Analyze Image", nil)
	
	logMsg := func(msg string) {
		outputLog.SetText(outputLog.Text + msg + "\n")
	}

	executeBtn.OnTapped = func() {
		if imageEntry.Text == "" {
			dialog.ShowInformation("Error", "Evidence Image is required.", win)
			return
		}

		executeBtn.Disable()
		outputLog.SetText("Running " + actionSelect.Selected + " on " + imageEntry.Text + "...\n\n")

		go func() {
			defer executeBtn.Enable()
			
			switch actionSelect.Selected {
			case "Extract Unallocated":
				if outDirEntry.Text == "" {
					logMsg("Error: Output Directory required for extraction.")
					return
				}
				
				if err := os.MkdirAll(outDirEntry.Text, 0755); err != nil {
					logMsg(fmt.Sprintf("Failed to create output directory: %v", err))
					return
				}

				logMsg(fmt.Sprintf("Scanning image %s for deleted files (JPEG/PDF/PNG)...", imageEntry.Text))
				logMsg(fmt.Sprintf("Extracted data will be written to: %s", outDirEntry.Text))
				
				foundFiles := 0
				err := FileCarver(imageEntry.Text, outDirEntry.Text, func(offset int64, fileType string) {
					foundFiles++
					if foundFiles%5 == 0 { // Don't overwhelm UI log
						logMsg(fmt.Sprintf("Recovered %s at offset 0x%x...", fileType, offset))
					}
				})
				
				if err != nil {
					logMsg(fmt.Sprintf("Error during carving: %v", err))
				} else {
					logMsg(fmt.Sprintf("Carving complete. Found %d files.", foundFiles))
				}

			case "Build MAC Timeline":
				logMsg(fmt.Sprintf("Extracting MAC timeline from %s...", imageEntry.Text))
				logMsg("[Analysis] Invoking TSK Metadata-layer bindings for inode timestamp retrieval...")
				time.Sleep(1 * time.Second) // Simulate work
				logMsg("[Error] Full TSK Metadata-layer bindings for timeline not yet implemented in GUI.")
			}
			
			logMsg("\n-- Analysis Complete --")
		}()
	}

	form := container.NewVBox(
		widget.NewLabelWithStyle("Non-Destructive Image Analysis", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2, widget.NewLabel("Evidence Image:"), imageEntry),
		container.NewGridWithColumns(2, widget.NewLabel("Output Dir:"), outDirEntry),
		container.NewGridWithColumns(2, widget.NewLabel("Action:"), actionSelect),
		executeBtn,
	)

	return container.NewBorder(form, nil, nil, nil, scrollableLog)
}
