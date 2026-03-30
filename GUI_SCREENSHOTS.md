# GUI Screenshots and Visual Guide

This document provides a visual description of the GUI interfaces (actual screenshots would be generated when running the applications).

## Fyne Desktop GUI

### Main Dashboard
```
┌────────────────────────────────────────────────────────────────┐
│ Diskimager Forensics Suite                          [⚙️][ℹ️]   │
│ Professional Digital Evidence Acquisition and Analysis         │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Quick Actions                                                  │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐    │
│  │   Imaging      │ │   Restore      │ │   Clone        │    │
│  │ Create forensic│ │ Restore images │ │ Direct disk-to-│    │
│  │ disk images... │ │ to disks...    │ │ disk cloning   │    │
│  │  [Create Image]│ │ [Restore Image]│ │  [Clone Disk]  │    │
│  └────────────────┘ └────────────────┘ └────────────────┘    │
│                                                                 │
│  ┌────────────────┐ ┌────────────────┐                        │
│  │   Search       │ │   Analyze      │                        │
│  │ Find files...  │ │ Deep analysis  │                        │
│  │  [Find Files]  │ │  [Analyze]     │                        │
│  └────────────────┘ └────────────────┘                        │
│                                                                 │
│  Recent Operations                              [Clear History]│
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ ✓ [2026-03-29 14:23] Image /dev/disk1 → evidence.e01    │ │
│  │ ✓ [2026-03-29 11:45] Restore backup.img → /dev/disk2    │ │
│  │ ✓ [2026-03-28 16:12] Image /dev/disk3 → cloud storage   │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
│  System Health (SMART Status)                       [Refresh]  │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ ✓ 💽 /dev/disk0 - System SSD    | Healthy | 35°C        │ │
│  │ ✓ 💾 /dev/disk1 - Data HDD      | Healthy | 42°C        │ │
│  │ ✓ 🔌 /dev/disk2 - USB Drive     | Healthy | N/A         │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
├────────────────────────────────────────────────────────────────┤
│ Ready                           v1.0.0        14:30:15 PST     │
└────────────────────────────────────────────────────────────────┘
```

### Imaging Wizard - Step 1: Source Selection
```
┌────────────────────────────────────────────────────────────────┐
│ Step 1: Source Selection                               [⬅️][➡️] │
│ Select the disk or image file you want to image                │
│ Progress: ▓░░░░░░░░░░░░░░░░░░░░ 17%                           │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Available Disks:                                               │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ 💽 System SSD                                             │ │
│  │    /dev/disk0                                             │ │
│  │    500 GB | Apple SSD Controller                          │ │
│  │    Healthy • SSD                                          │ │
│  ├──────────────────────────────────────────────────────────┤ │
│  │ 💾 Data HDD                                               │ │
│  │    /dev/disk1                         [SELECTED]          │ │
│  │    2.0 TB | WD Blue 2TB                                   │ │
│  │    Healthy • HDD                                          │ │
│  ├──────────────────────────────────────────────────────────┤ │
│  │ 🔌 USB Drive                                              │ │
│  │    /dev/disk2                                             │ │
│  │    1.0 TB | SanDisk Ultra                                 │ │
│  │    Healthy • USB                                          │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ────────────────────────────────────────────────                │
│                                                                 │
│  Or select a file:                                              │
│  ┌────────────────────────────────────────────┐ [📁 Browse...] │
│  │ /path/to/image.dd                          │                │
│  └────────────────────────────────────────────┘                │
│                                                                 │
├────────────────────────────────────────────────────────────────┤
│ [Cancel]                                 [⬅️ Back]  [Next ➡️]   │
└────────────────────────────────────────────────────────────────┘
```

### Imaging Wizard - Step 5: Execution
```
┌────────────────────────────────────────────────────────────────┐
│ Step 5: Execution                                      [⬅️][➡️] │
│ Imaging operation in progress - do not interrupt                │
│ Progress: ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░ 75%                             │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Phase: reading                                                 │
│  ████████████████████████████████████░░░░░░░ 75.3%            │
│  Speed: 125.42 MB/s                      ETA: 5m 23s           │
│  Status: Running (0 bad sectors)                                │
│                                                                 │
│  ────────────────────────────────────────────────                │
│                                                                 │
│  Operation Log:                                                 │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ [14:30:15] Opening source: /dev/disk1                     │ │
│  │ [14:30:15] Opening destination: evidence-001.e01          │ │
│  │ [14:30:16] Starting imaging operation                     │ │
│  │ [14:32:45] Progress: 25% complete                         │ │
│  │ [14:35:12] Progress: 50% complete                         │ │
│  │ [14:37:38] Progress: 75% complete                         │ │
│  │                                                            │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
│  [🛑 Cancel Operation]                                          │
│                                                                 │
├────────────────────────────────────────────────────────────────┤
│ [Cancel]                                 [⬅️ Back]  [Next ➡️]   │
└────────────────────────────────────────────────────────────────┘
```

## Web UI

### Dashboard View
```
┌────────────────────────────────────────────────────────────────────────┐
│ 🔷 Diskimager Forensics Suite                            [⚙️ Settings] │
│    Web-Based Digital Evidence Acquisition                               │
├────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐             │
│  │ 📊 Active Jobs│  │ 💾 Disks      │  │ ✅ Completed  │             │
│  │      2        │  │      3        │  │      15       │             │
│  │ Currently run │  │ Ready for use │  │ Total jobs    │             │
│  └───────────────┘  └───────────────┘  └───────────────┘             │
│                                                                          │
│  ┌─────────────────┐  ┌────────────────────────────────────────────┐  │
│  │ Available Disks │  │ Job Queue                                   │  │
│  ├─────────────────┤  ├────────────────────────────────────────────┤  │
│  │ 💽 System SSD   │  │ 🔵 Image Job                    ✓ Running  │  │
│  │    /dev/disk0   │  │    /dev/disk1 → evidence.e01               │  │
│  │    500 GB       │  │    E01 | 2.0 TB | 125.42 MB/s              │  │
│  │    Healthy • SSD│  │    ████████████░░░░░░░░ 75.3%              │  │
│  │                 │  │    reading | ETA: 5m 23s            [🗑️]   │  │
│  ├─────────────────┤  ├────────────────────────────────────────────┤  │
│  │ 💾 Data HDD     │  │ 🟡 Image Job                   ⏸️ Pending  │  │
│  │    /dev/disk1   │  │    /dev/disk2 → backup.e01                 │  │
│  │    2.0 TB       │  │    E01 | 1.0 TB                     [🗑️]   │  │
│  │    Healthy • HDD│  ├────────────────────────────────────────────┤  │
│  ├─────────────────┤  │ 🟢 Image Job                   ✅ Complete  │  │
│  │ 🔌 USB Drive    │  │    /dev/disk3 → archive.dd                 │  │
│  │    /dev/disk2   │  │    Raw | 500 GB | Hash: 8f3a...     [🗑️]   │  │
│  │    1.0 TB       │  └────────────────────────────────────────────┘  │
│  │    Healthy • USB│                                                   │
│  │                 │                                                   │
│  │  [🔄 Refresh]   │                                                   │
│  └─────────────────┘                                                   │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │ Progress Monitor - Job: 550e8400-e29b-41d4-a716-446655440000    │  │
│  ├─────────────────────────────────────────────────────────────────┤  │
│  │ 🟢 Real-time updates active                                      │  │
│  │                                                                   │  │
│  │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │  │
│  │ │ Status   │ │ Phase    │ │ Progress │ │ Speed    │           │  │
│  │ │ running  │ │ reading  │ │ 75.3%    │ │125.42MB/s│           │  │
│  │ └──────────┘ └──────────┘ └──────────┘ └──────────┘           │  │
│  │                                                                   │  │
│  │ 📊 Transfer Speed                                                │  │
│  │ 150 MB/s ┤                                                       │  │
│  │ 125 MB/s ┤    ╭─╮    ╭──╮                                       │  │
│  │ 100 MB/s ┤  ╭─╯ ╰────╯  ╰──╮                                    │  │
│  │  75 MB/s ┤──╯              ╰─────                               │  │
│  │  50 MB/s ┤                                                       │  │
│  │  25 MB/s ┤                                                       │  │
│  │   0 MB/s └─┬────┬────┬────┬────┬────                           │  │
│  │           14:30 14:31 14:32 14:33 14:34                         │  │
│  └─────────────────────────────────────────────────────────────────┘  │
│                                                                          │
├────────────────────────────────────────────────────────────────────────┤
│ Diskimager Forensics Suite v1.0.0          © 2026 AfterDark Systems   │
└────────────────────────────────────────────────────────────────────────┘
```

## Color Schemes

### Fyne Desktop (Default Theme)
- **Primary**: Fyne blue (#0A84FF)
- **Background**: System default (light/dark)
- **Text**: System default
- **Success**: Green (#34C759)
- **Warning**: Yellow (#FF9500)
- **Error**: Red (#FF3B30)
- **Info**: Blue (#007AFF)

### Web UI (Forensic Theme)
- **Primary**: Forensic Blue (#0ea5e9)
- **Success**: Green (#10b981)
- **Warning**: Yellow (#f59e0b)
- **Error**: Red (#ef4444)
- **Background**: Gray (#f3f4f6)
- **Text**: Dark Gray (#1f2937)
- **Borders**: Light Gray (#d1d5db)

## Icons Used

### Fyne Desktop
- Storage Icon (💽): Disks
- Check Icon (✓): Success/Healthy
- Warning Icon (⚠️): Warnings
- Error Icon (✗): Errors
- Clock Icon (🕐): Time
- Settings Icon (⚙️): Settings
- Info Icon (ℹ️): About

### Web UI (Lucide React)
- `Activity`: Jobs, operations
- `HardDrive`: Disks
- `CheckCircle`: Success
- `XCircle`: Failure
- `Clock`: Pending
- `Loader`: Loading/Running
- `Trash2`: Delete
- `Settings`: Configuration
- `AlertCircle`: Errors

## Responsive Breakpoints (Web UI)

```
Mobile    : <640px   (1 column)
Tablet    : 640-1024px (2 columns)
Desktop   : >1024px  (3 columns)
```

## Accessibility Features

### Fyne Desktop
- Keyboard navigation (Tab, Enter, Esc)
- Screen reader compatible
- High contrast support
- Keyboard shortcuts (Ctrl+I, Ctrl+R, Ctrl+Q)

### Web UI
- ARIA labels on all interactive elements
- Keyboard navigation
- Focus indicators
- Color contrast (WCAG AA compliant)
- Screen reader announcements for status changes

## Animation & Transitions

### Fyne Desktop
- Smooth wizard step transitions (fade)
- Progress bar animations (linear)
- Button hover states
- Dialog fade in/out

### Web UI
- Page transitions (fade)
- Progress bar smooth updates
- Speed graph real-time animation
- Hover states (scale, color)
- Loading spinners

## Print Layouts

### Forensic Reports (Generated from Summary)
```
┌─────────────────────────────────────────────┐
│ FORENSIC IMAGING REPORT                     │
│                                             │
│ Generated: 2026-03-29T14:40:00Z            │
│                                             │
│ CASE INFORMATION                            │
│ Case Number: CASE-2026-001                  │
│ Examiner: John Doe                          │
│ Evidence ID: LAPTOP-001                     │
│                                             │
│ SOURCE                                      │
│ Path: /dev/disk1                            │
│ Size: 2.0 TB                                │
│ Model: WD Blue 2TB                          │
│                                             │
│ DESTINATION                                 │
│ Path: evidence-001.e01                      │
│ Format: E01 (EnCase Expert Witness)         │
│                                             │
│ OPERATION SUMMARY                           │
│ Status: Completed Successfully              │
│ Start Time: 2026-03-29 14:30:15            │
│ End Time: 2026-03-29 14:40:00              │
│ Duration: 9m 45s                            │
│ Average Speed: 125.42 MB/s                  │
│                                             │
│ VERIFICATION                                │
│ Hash Algorithm: SHA256                      │
│ Hash Value:                                 │
│ 8f3ab4c9e7d2f1a6b5c8e9d0a1b2c3d4e5f6g7h8 │
│                                             │
│ Bad Sectors: 0                              │
│ Errors: None                                │
│                                             │
│ This report was generated by                │
│ Diskimager Forensics Suite v1.0.0          │
└─────────────────────────────────────────────┘
```

## Notes

To generate actual screenshots:

1. **Fyne Desktop**: Run `./diskimager ui` and use system screenshot tool
2. **Web UI**: Run backend + frontend, use browser screenshot or developer tools
3. **Reports**: Use "Generate Report" button in wizard step 6

All visual elements are designed for professional forensic use with attention to:
- Clear information hierarchy
- Status indicators
- Progress visualization
- Error communication
- Professional appearance
- Norton Ghost-style familiarity

---

© 2026 AfterDark Systems - Diskimager Forensics Suite
