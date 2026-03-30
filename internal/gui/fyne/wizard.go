package fyne

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// WizardStep represents a single step in a wizard
type WizardStep interface {
	// Title returns the step title
	Title() string

	// Description returns a brief description of the step
	Description() string

	// Content returns the UI content for this step
	Content() fyne.CanvasObject

	// Validate validates the current step
	Validate() error

	// OnEnter is called when entering this step
	OnEnter()

	// OnExit is called when leaving this step
	OnExit()

	// CanProgress returns whether user can proceed to next step
	CanProgress() bool
}

// Wizard manages a multi-step wizard interface
type Wizard struct {
	steps       []WizardStep
	currentStep int
	onComplete  func()
	onCancel    func()

	window        fyne.Window
	content       *fyne.Container
	stepLabel     *widget.Label
	descLabel     *widget.Label
	stepContainer *fyne.Container
	backButton    *widget.Button
	nextButton    *widget.Button
	cancelButton  *widget.Button
	progressBar   *widget.ProgressBar
}

// NewWizard creates a new wizard with the given steps
func NewWizard(window fyne.Window, steps []WizardStep, onComplete, onCancel func()) *Wizard {
	w := &Wizard{
		steps:       steps,
		currentStep: 0,
		onComplete:  onComplete,
		onCancel:    onCancel,
		window:      window,
	}

	w.buildUI()
	return w
}

// buildUI constructs the wizard UI
func (w *Wizard) buildUI() {
	// Header with step info
	w.stepLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	w.descLabel = widget.NewLabel("")
	w.descLabel.Wrapping = fyne.TextWrapWord

	w.progressBar = widget.NewProgressBar()

	header := container.NewVBox(
		w.stepLabel,
		w.descLabel,
		w.progressBar,
		widget.NewSeparator(),
	)

	// Step content container
	w.stepContainer = container.NewMax()

	// Navigation buttons
	w.backButton = widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() {
		w.goBack()
	})

	w.nextButton = widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() {
		w.goNext()
	})
	w.nextButton.Importance = widget.HighImportance

	w.cancelButton = widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		if w.onCancel != nil {
			w.onCancel()
		}
	})

	buttonBar := container.NewBorder(
		nil, nil,
		w.cancelButton,
		container.NewHBox(w.backButton, w.nextButton),
	)

	// Main layout
	w.content = container.NewBorder(
		header,
		buttonBar,
		nil, nil,
		container.NewPadded(w.stepContainer),
	)

	w.updateStep()
}

// Content returns the wizard's UI content
func (w *Wizard) Content() fyne.CanvasObject {
	return w.content
}

// updateStep updates the UI for the current step
func (w *Wizard) updateStep() {
	if w.currentStep < 0 || w.currentStep >= len(w.steps) {
		return
	}

	step := w.steps[w.currentStep]
	step.OnEnter()

	// Update header
	w.stepLabel.SetText(step.Title())
	w.descLabel.SetText(step.Description())

	// Update progress
	progress := float64(w.currentStep) / float64(len(w.steps))
	w.progressBar.SetValue(progress)

	// Update content
	w.stepContainer.Objects = []fyne.CanvasObject{step.Content()}
	w.stepContainer.Refresh()

	// Update buttons
	if w.currentStep == 0 {
		w.backButton.Disable()
	} else {
		w.backButton.Enable()
	}

	if w.currentStep == len(w.steps)-1 {
		w.nextButton.SetText("Finish")
		w.nextButton.SetIcon(theme.ConfirmIcon())
	} else {
		w.nextButton.SetText("Next")
		w.nextButton.SetIcon(theme.NavigateNextIcon())
	}

	if step.CanProgress() {
		w.nextButton.Enable()
	} else {
		w.nextButton.Disable()
	}

	w.backButton.Refresh()
	w.nextButton.Refresh()
}

// goNext moves to the next step
func (w *Wizard) goNext() {
	if w.currentStep >= len(w.steps) {
		return
	}

	step := w.steps[w.currentStep]

	// Validate current step
	if err := step.Validate(); err != nil {
		w.window.Canvas().Unfocus()
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Validation Error",
			Content: err.Error(),
		})
		return
	}

	step.OnExit()

	// If last step, complete wizard
	if w.currentStep == len(w.steps)-1 {
		if w.onComplete != nil {
			w.onComplete()
		}
		return
	}

	w.currentStep++
	w.updateStep()
}

// goBack moves to the previous step
func (w *Wizard) goBack() {
	if w.currentStep <= 0 {
		return
	}

	w.steps[w.currentStep].OnExit()
	w.currentStep--
	w.updateStep()
}

// EnableNext enables or disables the next button
func (w *Wizard) EnableNext(enabled bool) {
	if enabled {
		w.nextButton.Enable()
	} else {
		w.nextButton.Disable()
	}
	w.nextButton.Refresh()
}

// CurrentStep returns the current step index
func (w *Wizard) CurrentStep() int {
	return w.currentStep
}

// SetStep sets the current step (useful for skipping steps)
func (w *Wizard) SetStep(step int) {
	if step < 0 || step >= len(w.steps) {
		return
	}

	w.steps[w.currentStep].OnExit()
	w.currentStep = step
	w.updateStep()
}

// BaseStep provides a default implementation for WizardStep
type BaseStep struct {
	title       string
	description string
	content     fyne.CanvasObject
	validator   func() error
	canProgress func() bool
	onEnter     func()
	onExit      func()
}

// NewBaseStep creates a new base step
func NewBaseStep(title, description string, content fyne.CanvasObject) *BaseStep {
	return &BaseStep{
		title:       title,
		description: description,
		content:     content,
		validator:   func() error { return nil },
		canProgress: func() bool { return true },
		onEnter:     func() {},
		onExit:      func() {},
	}
}

// Title returns the step title
func (s *BaseStep) Title() string {
	return s.title
}

// Description returns the step description
func (s *BaseStep) Description() string {
	return s.description
}

// Content returns the step content
func (s *BaseStep) Content() fyne.CanvasObject {
	return s.content
}

// Validate validates the step
func (s *BaseStep) Validate() error {
	if s.validator != nil {
		return s.validator()
	}
	return nil
}

// OnEnter is called when entering the step
func (s *BaseStep) OnEnter() {
	if s.onEnter != nil {
		s.onEnter()
	}
}

// OnExit is called when leaving the step
func (s *BaseStep) OnExit() {
	if s.onExit != nil {
		s.onExit()
	}
}

// CanProgress returns whether the step can proceed
func (s *BaseStep) CanProgress() bool {
	if s.canProgress != nil {
		return s.canProgress()
	}
	return true
}

// SetValidator sets the validation function
func (s *BaseStep) SetValidator(validator func() error) {
	s.validator = validator
}

// SetCanProgress sets the can progress function
func (s *BaseStep) SetCanProgress(canProgress func() bool) {
	s.canProgress = canProgress
}

// SetOnEnter sets the on enter function
func (s *BaseStep) SetOnEnter(onEnter func()) {
	s.onEnter = onEnter
}

// SetOnExit sets the on exit function
func (s *BaseStep) SetOnExit(onExit func()) {
	s.onExit = onExit
}
