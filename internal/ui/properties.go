package ui

import (
	"image/color"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/alexwbaule/turing-screen/internal/theme"
	"github.com/alexwbaule/turing-screen/internal/utils"
)

type PropertiesPanel struct {
	container      *fyne.Container
	app            *EditorApp
	selectedWidget fyne.CanvasObject
	headerLabel    *widget.Label

	refreshMu      sync.Mutex
	refreshTimer   *time.Timer
	updating       bool   // true while Update() is populating fields — suppresses OnChanged callbacks
	lastScrollType string // "text" | "graph" | "radial" | "chart" | "" — avoids VBox rebuild on same-type clicks

	sensorSelect  *widget.Select
	typeSelect    *widget.Select
	textFields    fyne.CanvasObject // *widget.Accordion
	graphFields   fyne.CanvasObject // *widget.Accordion
	radialFields  fyne.CanvasObject // *widget.Accordion
	chartFields   fyne.CanvasObject // *widget.Accordion
	commonFields  *fyne.Container
	placeholder *widget.Label
	scroll      *container.Scroll

	yamlPathLabel *widget.Label
	xEntry        *widget.Entry
	yEntry        *widget.Entry
	deleteButton  *widget.Button

	textContentEntry *widget.Entry
	fontSelector     *widget.Select
	fontSizeEntry    *widget.Entry
	fontColorEntry   *widget.Entry
	fontColorBtn     *widget.Button
	bgColorEntry     *widget.Entry
	bgColorBtn       *widget.Button
	alignSelect      *widget.Select
	formatSelect     *widget.Select
	showUnitCheck    *widget.Check

	graphWidthEntry          *widget.Entry
	graphHeightEntry         *widget.Entry
	graphMinEntry            *widget.Entry
	graphMaxEntry            *widget.Entry
	graphBarColorEntry       *widget.Entry
	graphBarColorBtn         *widget.Button
	graphBgColorEntry        *widget.Entry
	graphBgColorBtn          *widget.Button
	graphGradientColorEntry  *widget.Entry
	graphGradientColorBtn    *widget.Button
	graphOutlineCheck        *widget.Check
	graphStepsEntry          *widget.Entry
	graphStepGapEntry        *widget.Entry
	graphBlockWidthEntry     *widget.Entry
	graphCornerRadiusEntry   *widget.Entry
	graphBorderWidthEntry    *widget.Entry
	graphRevertValueCheck    *widget.Check
	graphDirectionSelect     *widget.Select

	radialRadiusEntry        *widget.Entry
	radialWidthEntry         *widget.Entry
	radialMinEntry           *widget.Entry
	radialMaxEntry           *widget.Entry
	radialStartEntry         *widget.Entry
	radialEndEntry           *widget.Entry
	radialStepsEntry         *widget.Entry
	radialSepEntry           *widget.Entry
	radialBlockAngleEntry    *widget.Entry
	radialClockCheck         *widget.Check
	radialRoundCheck         *widget.Check
	radialRevertCheck        *widget.Check
	radialRevertValueCheck   *widget.Check
	radialBarColorEntry      *widget.Entry
	radialBarColorBtn        *widget.Button
	radialBgColorEntry       *widget.Entry
	radialBgColorBtn         *widget.Button
	radialGradientColorEntry *widget.Entry
	radialGradientColorBtn   *widget.Button
	radialShowTextCheck      *widget.Check
	radialShowUnitCheck      *widget.Check
	radialFontSelector       *widget.Select
	radialFontColorEntry     *widget.Entry
	radialFontColorBtn       *widget.Button

	chartWidthEntry     *widget.Entry
	chartHeightEntry    *widget.Entry
	chartMinEntry       *widget.Entry
	chartMaxEntry       *widget.Entry
	chartColWidthEntry  *widget.Entry
	chartColGapEntry    *widget.Entry
	chartFillColorEntry *widget.Entry
	chartFillColorBtn   *widget.Button
	chartLineColorEntry *widget.Entry
	chartLineColorBtn   *widget.Button
	chartBorderEntry    *widget.Entry
}

// setLabel/setEntry/setSelect/setCheck skip the Fyne setter when the value is
// already correct. widget.Label.SetText always fires Refresh regardless; the
// others have internal guards but we add our own to avoid lock contention on
// the hot path through Update().
func setLabel(l *widget.Label, text string) {
	if l.Text != text {
		l.SetText(text)
	}
}
func setEntry(e *widget.Entry, text string) {
	if e.Text != text {
		e.SetText(text)
	}
}
func setSelect(s *widget.Select, text string) {
	if s.Selected != text {
		s.SetSelected(text)
	}
}
func setCheck(c *widget.Check, checked bool) {
	if c.Checked != checked {
		c.SetChecked(checked)
	}
}

// propWindow returns the properties window if it exists, otherwise falls back to
// the active editor window. Used as the dialog parent so that color picker dialogs
// appear anchored to the correct window.
func (p *PropertiesPanel) propWindow() fyne.Window {
	if p.app.propertiesWindow != nil {
		return p.app.propertiesWindow
	}
	return p.app.activeWindow()
}

// scheduleRefresh debounces visual refreshes so that rapid OnChanged events
// (e.g. every keystroke in a text entry) only trigger one buildImage() call
// after 150ms of inactivity instead of one per keystroke.
func (p *PropertiesPanel) scheduleRefresh(w fyne.Widget) {
	if p.updating {
		return
	}
	p.refreshMu.Lock()
	defer p.refreshMu.Unlock()
	if p.refreshTimer != nil {
		p.refreshTimer.Stop()
	}
	p.refreshTimer = time.AfterFunc(300*time.Millisecond, func() {
		fyne.Do(func() {
			w.Refresh()
			w.Resize(w.MinSize())
		})
	})
}

func buildProperties(app *EditorApp) *PropertiesPanel {
	p := &PropertiesPanel{app: app}
	p.headerLabel = widget.NewLabelWithStyle("Propriedades", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Sensor and Type selects
	p.sensorSelect = widget.NewSelect(sensorOptions, func(s string) {
		p.onSensorChanged(s)
	})
	p.typeSelect = widget.NewSelect(widgetTypeOptions, func(s string) {
		p.onTypeChanged(s)
	})

	availableFonts := app.fontCache.AvailableFonts()

	// === COMMON FIELDS ===
	p.yamlPathLabel = widget.NewLabel("")
	p.yamlPathLabel.Wrapping = fyne.TextWrapWord

	p.xEntry = widget.NewEntry()
	p.xEntry.OnChanged = func(s string) {
		if v, err := strconv.Atoi(s); err == nil {
			p.updateX(v)
		}
	}
	p.yEntry = widget.NewEntry()
	p.yEntry.OnChanged = func(s string) {
		if v, err := strconv.Atoi(s); err == nil {
			p.updateY(v)
		}
	}
	p.deleteButton = widget.NewButtonWithIcon("Excluir Componente", fyneTheme.DeleteIcon(), func() {
		app.DeleteSelectedWidget()
	})
	p.deleteButton.Importance = widget.DangerImportance

	p.commonFields = container.NewVBox(
		widget.NewLabel("Caminho YAML:"),
		p.yamlPathLabel,
		widget.NewSeparator(),
		widget.NewLabel("Posição X:"),
		p.xEntry,
		widget.NewLabel("Posição Y:"),
		p.yEntry,
		widget.NewSeparator(),
		p.deleteButton,
	)

	// === TEXT FIELDS ===
	p.textContentEntry = widget.NewEntry()
	p.textContentEntry.OnChanged = func(s string) {
		if dw, ok := p.selectedWidget.(*DraggableWidget); ok {
			if strings.HasPrefix(dw.YAMLPath, "static_texts.") {
				dw.TextData.Text = s
			} else {
				dw.TextData.Placeholder = s
			}
			p.scheduleRefresh(dw)
		}
	}
	p.fontSelector = widget.NewSelect(availableFonts, func(s string) {
		if dw, ok := p.selectedWidget.(*DraggableWidget); ok {
			dw.TextData.Font = s
			p.scheduleRefresh(dw)
		}
	})
	p.fontSizeEntry = widget.NewEntry()
	p.fontSizeEntry.OnChanged = func(s string) {
		if dw, ok := p.selectedWidget.(*DraggableWidget); ok {
			if v, err := strconv.Atoi(s); err == nil {
				dw.TextData.FontSize = v
				p.scheduleRefresh(dw)
			}
		}
	}
	p.fontColorEntry = widget.NewEntry()
	p.fontColorEntry.OnChanged = func(s string) {
		if dw, ok := p.selectedWidget.(*DraggableWidget); ok {
			dw.TextData.FontColor = s
			p.scheduleRefresh(dw)
		}
	}
	p.fontColorBtn = widget.NewButton("Picker", func() {
		if dw, ok := p.selectedWidget.(*DraggableWidget); ok {
			picker := dialog.NewColorPicker("Cor da Fonte", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.fontColorEntry.SetText(val)
				dw.TextData.FontColor = val
				dw.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.bgColorEntry = widget.NewEntry()
	p.bgColorEntry.OnChanged = func(s string) {
		if dw, ok := p.selectedWidget.(*DraggableWidget); ok {
			dw.TextData.BackgroundColor = s
			p.scheduleRefresh(dw)
		}
	}
	p.bgColorBtn = widget.NewButton("Picker", func() {
		if dw, ok := p.selectedWidget.(*DraggableWidget); ok {
			picker := dialog.NewColorPicker("Cor de Fundo", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.bgColorEntry.SetText(val)
				dw.TextData.BackgroundColor = val
				dw.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.alignSelect = widget.NewSelect([]string{"LEFT", "CENTER", "RIGHT"}, func(s string) {
		if dw, ok := p.selectedWidget.(*DraggableWidget); ok {
			dw.TextData.Align = s
			// Reposition widget based on new alignment
			posX := float32(dw.TextData.X)
			widgetWidth := dw.Size().Width
			switch s {
			case "CENTER":
				posX = float32(dw.TextData.X) - widgetWidth/2
			case "RIGHT":
				posX = float32(dw.TextData.X) - widgetWidth
			}
			if posX < 0 {
				posX = 0
			}
			dw.Move(fyne.NewPos(posX, float32(dw.TextData.Y)))
		}
	})
	p.formatSelect = widget.NewSelect([]string{"SHORT", "MEDIUM", "LONG", "FULL"}, func(s string) {
		if dw, ok := p.selectedWidget.(*DraggableWidget); ok {
			dw.TextData.Format = s
		}
	})
	p.showUnitCheck = widget.NewCheck("Mostrar Unidade", func(checked bool) {
		if dw, ok := p.selectedWidget.(*DraggableWidget); ok {
			dw.TextData.ShowUnit = checked
		}
	})
	textAcc := widget.NewAccordion(
		widget.NewAccordionItem("Conteúdo", container.NewVBox(
			widget.NewLabel("Texto:"),
			p.textContentEntry,
		)),
		widget.NewAccordionItem("Fonte", container.NewVBox(
			widget.NewLabel("Fonte:"),
			p.fontSelector,
			widget.NewLabel("Tamanho:"),
			p.fontSizeEntry,
			widget.NewLabel("Cor da Fonte (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.fontColorBtn, p.fontColorEntry),
			widget.NewLabel("Cor de Fundo (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.bgColorBtn, p.bgColorEntry),
		)),
		widget.NewAccordionItem("Layout", container.NewVBox(
			widget.NewLabel("Alinhamento:"),
			p.alignSelect,
			widget.NewLabel("Formato (datas):"),
			p.formatSelect,
			p.showUnitCheck,
		)),
	)
	textAcc.MultiOpen = true
	textAcc.Open(0)
	p.textFields = textAcc

	// === GRAPH FIELDS ===
	makeIntEntry := func(onChange func(int)) *widget.Entry {
		e := widget.NewEntry()
		e.OnChanged = func(s string) {
			if v, err := strconv.Atoi(s); err == nil {
				onChange(v)
			}
		}
		return e
	}
	p.graphWidthEntry = makeIntEntry(func(v int) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.Width = v
			p.scheduleRefresh(dg)
		}
	})
	p.graphHeightEntry = makeIntEntry(func(v int) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.Height = v
			p.scheduleRefresh(dg)
		}
	})
	p.graphMinEntry = makeIntEntry(func(v int) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.MinValue = v
		}
	})
	p.graphMaxEntry = makeIntEntry(func(v int) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.MaxValue = v
		}
	})
	p.graphBarColorEntry = widget.NewEntry()
	p.graphBarColorEntry.OnChanged = func(s string) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.BarColor = s
			p.scheduleRefresh(dg)
		}
	}
	p.graphBarColorBtn = widget.NewButton("Picker", func() {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			picker := dialog.NewColorPicker("Cor da Barra", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.graphBarColorEntry.SetText(val)
				dg.GraphData.BarColor = val
				dg.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.graphBgColorEntry = widget.NewEntry()
	p.graphBgColorEntry.OnChanged = func(s string) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.BackgroundColor = s
			p.scheduleRefresh(dg)
		}
	}
	p.graphBgColorBtn = widget.NewButton("Picker", func() {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			picker := dialog.NewColorPicker("Cor de Fundo", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.graphBgColorEntry.SetText(val)
				dg.GraphData.BackgroundColor = val
				dg.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.graphOutlineCheck = widget.NewCheck("Outline", func(checked bool) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.BarOutline = checked
			p.scheduleRefresh(dg)
		}
	})
	p.graphStepsEntry = makeIntEntry(func(v int) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.Steps = v
			p.scheduleRefresh(dg)
		}
	})
	p.graphStepGapEntry = makeIntEntry(func(v int) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.StepGap = v
			p.scheduleRefresh(dg)
		}
	})
	p.graphGradientColorEntry = widget.NewEntry()
	p.graphGradientColorEntry.OnChanged = func(s string) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.GradientColor = s
			p.scheduleRefresh(dg)
		}
	}
	p.graphGradientColorBtn = widget.NewButton("Picker", func() {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			picker := dialog.NewColorPicker("Gradiente", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.graphGradientColorEntry.SetText(val)
				dg.GraphData.GradientColor = val
				dg.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.graphCornerRadiusEntry = makeIntEntry(func(v int) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.CornerRadius = v
			p.scheduleRefresh(dg)
		}
	})
	p.graphBorderWidthEntry = makeIntEntry(func(v int) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.BorderWidth = v
			p.scheduleRefresh(dg)
		}
	})
	p.graphBlockWidthEntry = makeIntEntry(func(v int) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.BlockWidth = v
			p.scheduleRefresh(dg)
		}
	})
	p.graphRevertValueCheck = widget.NewCheck("Inverter Valor", func(checked bool) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.RevertValue = checked
			p.scheduleRefresh(dg)
		}
	})
	p.graphDirectionSelect = widget.NewSelect([]string{"left", "right", "up", "down"}, func(s string) {
		if dg, ok := p.selectedWidget.(*DraggableGraph); ok {
			dg.GraphData.Direction = s
			p.scheduleRefresh(dg)
		}
	})
	graphAcc := widget.NewAccordion(
		widget.NewAccordionItem("Dimensões", container.NewVBox(
			widget.NewLabel("Largura:"), p.graphWidthEntry,
			widget.NewLabel("Altura:"), p.graphHeightEntry,
			widget.NewLabel("Direção:"), p.graphDirectionSelect,
		)),
		widget.NewAccordionItem("Aparência", container.NewVBox(
			widget.NewLabel("Cor da Barra (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.graphBarColorBtn, p.graphBarColorEntry),
			widget.NewLabel("Cor de Fundo (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.graphBgColorBtn, p.graphBgColorEntry),
			widget.NewLabel("Gradiente (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.graphGradientColorBtn, p.graphGradientColorEntry),
			p.graphOutlineCheck,
			p.graphRevertValueCheck,
		)),
		widget.NewAccordionItem("Valores", container.NewVBox(
			widget.NewLabel("Mín / Máx:"),
			container.NewGridWithColumns(2, p.graphMinEntry, p.graphMaxEntry),
		)),
		widget.NewAccordionItem("Avançado", container.NewVBox(
			widget.NewLabel("Segmentos / Gap:"),
			container.NewGridWithColumns(2, p.graphStepsEntry, p.graphStepGapEntry),
			widget.NewLabel("Largura do Bloco:"), p.graphBlockWidthEntry,
			widget.NewLabel("Raio do Canto:"), p.graphCornerRadiusEntry,
			widget.NewLabel("Espessura da Borda:"), p.graphBorderWidthEntry,
		)),
	)
	graphAcc.MultiOpen = true
	graphAcc.Open(0)
	p.graphFields = graphAcc

	// === RADIAL FIELDS ===
	p.radialRadiusEntry = makeIntEntry(func(v int) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.Radius = v
			p.scheduleRefresh(dr)
		}
	})
	p.radialWidthEntry = makeIntEntry(func(v int) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.Width = v
			p.scheduleRefresh(dr)
		}
	})
	p.radialMinEntry = makeIntEntry(func(v int) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.MinValue = v
		}
	})
	p.radialMaxEntry = makeIntEntry(func(v int) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.MaxValue = v
		}
	})
	p.radialStartEntry = makeIntEntry(func(v int) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.AngleStart = v
			p.scheduleRefresh(dr)
		}
	})
	p.radialEndEntry = makeIntEntry(func(v int) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.AngleEnd = v
			p.scheduleRefresh(dr)
		}
	})
	p.radialStepsEntry = makeIntEntry(func(v int) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.AngleSteps = v
			p.scheduleRefresh(dr)
		}
	})
	p.radialSepEntry = makeIntEntry(func(v int) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.AngleSep = v
			p.scheduleRefresh(dr)
		}
	})
	p.radialClockCheck = widget.NewCheck("Horário", func(checked bool) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.Clockwise = checked
			p.scheduleRefresh(dr)
		}
	})
	p.radialBarColorEntry = widget.NewEntry()
	p.radialBarColorEntry.OnChanged = func(s string) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.BarColor = s
			p.scheduleRefresh(dr)
		}
	}
	p.radialBarColorBtn = widget.NewButton("Picker", func() {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			picker := dialog.NewColorPicker("Cor do Arco", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.radialBarColorEntry.SetText(val)
				dr.RadialData.BarColor = val
				dr.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.radialBgColorEntry = widget.NewEntry()
	p.radialBgColorEntry.OnChanged = func(s string) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.BackgroundColor = s
			p.scheduleRefresh(dr)
		}
	}
	p.radialBgColorBtn = widget.NewButton("Picker", func() {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			picker := dialog.NewColorPicker("Cor de Fundo", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.radialBgColorEntry.SetText(val)
				dr.RadialData.BackgroundColor = val
				dr.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.radialShowTextCheck = widget.NewCheck("Mostrar Texto", func(checked bool) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.ShowText = checked
			p.scheduleRefresh(dr)
		}
	})
	p.radialShowUnitCheck = widget.NewCheck("Mostrar Unidade", func(checked bool) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.ShowUnit = checked
			p.scheduleRefresh(dr)
		}
	})
	p.radialFontSelector = widget.NewSelect(availableFonts, func(s string) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.Font = s
			p.scheduleRefresh(dr)
		}
	})
	p.radialFontColorEntry = widget.NewEntry()
	p.radialFontColorEntry.OnChanged = func(s string) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.FontColor = s
			p.scheduleRefresh(dr)
		}
	}
	p.radialFontColorBtn = widget.NewButton("Picker", func() {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			picker := dialog.NewColorPicker("Cor da Fonte", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.radialFontColorEntry.SetText(val)
				dr.RadialData.FontColor = val
				dr.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.radialGradientColorEntry = widget.NewEntry()
	p.radialGradientColorEntry.OnChanged = func(s string) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.GradientColor = s
			p.scheduleRefresh(dr)
		}
	}
	p.radialGradientColorBtn = widget.NewButton("Picker", func() {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			picker := dialog.NewColorPicker("Gradiente", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.radialGradientColorEntry.SetText(val)
				dr.RadialData.GradientColor = val
				dr.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.radialBlockAngleEntry = makeIntEntry(func(v int) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.BlockAngle = v
			p.scheduleRefresh(dr)
		}
	})
	p.radialRoundCheck = widget.NewCheck("Pontas Arredondadas", func(checked bool) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.Round = checked
			p.scheduleRefresh(dr)
		}
	})
	p.radialRevertCheck = widget.NewCheck("Inverter Cores (Revert)", func(checked bool) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.Revert = checked
			p.scheduleRefresh(dr)
		}
	})
	p.radialRevertValueCheck = widget.NewCheck("Inverter Valor", func(checked bool) {
		if dr, ok := p.selectedWidget.(*DraggableRadial); ok {
			dr.RadialData.RevertValue = checked
			p.scheduleRefresh(dr)
		}
	})
	radialAcc := widget.NewAccordion(
		widget.NewAccordionItem("Geometria", container.NewVBox(
			widget.NewLabel("Raio:"), p.radialRadiusEntry,
			widget.NewLabel("Espessura:"), p.radialWidthEntry,
			widget.NewLabel("Mín / Máx:"),
			container.NewGridWithColumns(2, p.radialMinEntry, p.radialMaxEntry),
		)),
		widget.NewAccordionItem("Ângulos", container.NewVBox(
			widget.NewLabel("Início / Fim:"),
			container.NewGridWithColumns(2, p.radialStartEntry, p.radialEndEntry),
			widget.NewLabel("Segmentos / Separação:"),
			container.NewGridWithColumns(2, p.radialStepsEntry, p.radialSepEntry),
			widget.NewLabel("Ângulo do Bloco:"), p.radialBlockAngleEntry,
			p.radialClockCheck,
			p.radialRoundCheck,
			p.radialRevertCheck,
			p.radialRevertValueCheck,
		)),
		widget.NewAccordionItem("Aparência", container.NewVBox(
			widget.NewLabel("Cor do Arco (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.radialBarColorBtn, p.radialBarColorEntry),
			widget.NewLabel("Cor de Fundo (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.radialBgColorBtn, p.radialBgColorEntry),
			widget.NewLabel("Gradiente (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.radialGradientColorBtn, p.radialGradientColorEntry),
		)),
		widget.NewAccordionItem("Texto", container.NewVBox(
			p.radialShowTextCheck,
			p.radialShowUnitCheck,
			widget.NewLabel("Fonte:"), p.radialFontSelector,
			widget.NewLabel("Cor da Fonte (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.radialFontColorBtn, p.radialFontColorEntry),
		)),
	)
	radialAcc.MultiOpen = true
	radialAcc.Open(0)
	p.radialFields = radialAcc

	// === CHART FIELDS ===
	p.chartWidthEntry = makeIntEntry(func(v int) {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			dc.ChartData.Width = v
			p.scheduleRefresh(dc)
		}
	})
	p.chartHeightEntry = makeIntEntry(func(v int) {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			dc.ChartData.Height = v
			p.scheduleRefresh(dc)
		}
	})
	p.chartMinEntry = makeIntEntry(func(v int) {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			dc.ChartData.MinValue = v
		}
	})
	p.chartMaxEntry = makeIntEntry(func(v int) {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			dc.ChartData.MaxValue = v
		}
	})
	p.chartColWidthEntry = makeIntEntry(func(v int) {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			dc.ChartData.ColumnWidth = v
			p.scheduleRefresh(dc)
		}
	})
	p.chartColGapEntry = makeIntEntry(func(v int) {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			dc.ChartData.ColumnGap = v
			p.scheduleRefresh(dc)
		}
	})
	p.chartFillColorEntry = widget.NewEntry()
	p.chartFillColorEntry.OnChanged = func(s string) {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			dc.ChartData.FillColor = s
			p.scheduleRefresh(dc)
		}
	}
	p.chartFillColorBtn = widget.NewButton("Picker", func() {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			picker := dialog.NewColorPicker("Cor Preenchimento", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.chartFillColorEntry.SetText(val)
				dc.ChartData.FillColor = val
				dc.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.chartLineColorEntry = widget.NewEntry()
	p.chartLineColorEntry.OnChanged = func(s string) {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			dc.ChartData.LineColor = s
			p.scheduleRefresh(dc)
		}
	}
	p.chartLineColorBtn = widget.NewButton("Picker", func() {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			picker := dialog.NewColorPicker("Cor da Linha", "", func(c color.Color) {
				val := utils.FormatColor(c)
				p.chartLineColorEntry.SetText(val)
				dc.ChartData.LineColor = val
				dc.Refresh()
			}, p.propWindow())
			picker.Show()
		}
	})
	p.chartBorderEntry = makeIntEntry(func(v int) {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			dc.ChartData.BorderWidth = v
			p.scheduleRefresh(dc)
		}
	})
	chartAcc := widget.NewAccordion(
		widget.NewAccordionItem("Dimensões", container.NewVBox(
			widget.NewLabel("Largura:"), p.chartWidthEntry,
			widget.NewLabel("Altura:"), p.chartHeightEntry,
		)),
		widget.NewAccordionItem("Valores", container.NewVBox(
			widget.NewLabel("Mín / Máx:"),
			container.NewGridWithColumns(2, p.chartMinEntry, p.chartMaxEntry),
		)),
		widget.NewAccordionItem("Aparência", container.NewVBox(
			widget.NewLabel("Largura Coluna:"), p.chartColWidthEntry,
			widget.NewLabel("Gap Coluna:"), p.chartColGapEntry,
			widget.NewLabel("Cor Preenchimento (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.chartFillColorBtn, p.chartFillColorEntry),
			widget.NewLabel("Cor da Linha (R,G,B):"),
			container.NewBorder(nil, nil, nil, p.chartLineColorBtn, p.chartLineColorEntry),
			widget.NewLabel("Borda:"), p.chartBorderEntry,
		)),
	)
	chartAcc.MultiOpen = true
	chartAcc.Open(0)
	p.chartFields = chartAcc

	// === PLACEHOLDER ===
	p.placeholder = widget.NewLabel("Selecione um componente para editar suas propriedades.")
	p.placeholder.Wrapping = fyne.TextWrapWord
	p.placeholder.Alignment = fyne.TextAlignCenter

	p.scroll = container.NewVScroll(p.placeholder)
	p.scroll.SetMinSize(fyne.NewSize(320, 1))

	header := container.NewVBox(
		p.headerLabel,
		widget.NewSeparator(),
		widget.NewLabel("Sensor:"),
		p.sensorSelect,
		widget.NewLabel("Tipo:"),
		p.typeSelect,
		widget.NewSeparator(),
	)

	// Scroll goes directly as center of Border — gets all remaining space, scrollbar works.
	p.container = container.NewBorder(
		header,
		nil, nil, nil,
		p.scroll,
	)
	p.Update(nil)
	return p
}

func (p *PropertiesPanel) updateX(v int) {
	if p.updating {
		return
	}
	switch w := p.selectedWidget.(type) {
	case *DraggableWidget:
		w.TextData.X = v
		// Reposition based on alignment
		posX := float32(v)
		widgetWidth := w.Size().Width
		align := strings.ToUpper(w.TextData.Align)
		switch align {
		case "CENTER":
			posX = float32(v) - widgetWidth/2
		case "RIGHT":
			posX = float32(v) - widgetWidth
		}
		if posX < 0 {
			posX = 0
		}
		w.Move(fyne.NewPos(posX, float32(w.TextData.Y)))
	case *DraggableGraph:
		w.GraphData.X = v
		w.Move(fyne.NewPos(float32(v), float32(w.GraphData.Y)))
	case *DraggableRadial:
		w.RadialData.X = v
		r := float32(w.RadialData.Radius)
		w.Move(fyne.NewPos(float32(v)-r, float32(w.RadialData.Y)-r))
	case *DraggableChart:
		w.ChartData.X = v
		w.Move(fyne.NewPos(float32(v), float32(w.ChartData.Y)))
	}
}

func (p *PropertiesPanel) updateY(v int) {
	if p.updating {
		return
	}
	switch w := p.selectedWidget.(type) {
	case *DraggableWidget:
		w.TextData.Y = v
		w.Move(fyne.NewPos(float32(w.TextData.X), float32(v)))
	case *DraggableGraph:
		w.GraphData.Y = v
		w.Move(fyne.NewPos(float32(w.GraphData.X), float32(v)))
	case *DraggableRadial:
		w.RadialData.Y = v
		r := float32(w.RadialData.Radius)
		w.Move(fyne.NewPos(float32(w.RadialData.X)-r, float32(v)-r))
	case *DraggableChart:
		w.ChartData.Y = v
		w.Move(fyne.NewPos(float32(w.ChartData.X), float32(v)))
	}
}

func (p *PropertiesPanel) Update(obj fyne.CanvasObject) {
	p.selectedWidget = obj

	if obj == nil {
		log.Printf("[PropertiesPanel] Update(nil) — mostrando placeholder")
		p.headerLabel.SetText("Propriedades")
		if p.lastScrollType != "" {
			p.lastScrollType = ""
			p.scroll.Content = p.placeholder
			p.scroll.Refresh()
		}
		return
	}

	log.Printf("[PropertiesPanel] Update(%T) — mostrando propriedades", obj)
	p.updating = true
	defer func() { p.updating = false }()

	var yamlPath string
	switch v := obj.(type) {
	case *DraggableWidget:
		yamlPath = v.YAMLPath
	case *DraggableGraph:
		yamlPath = v.YAMLPath
	case *DraggableRadial:
		yamlPath = v.YAMLPath
	case *DraggableChart:
		yamlPath = v.YAMLPath
	}

	if strings.HasPrefix(yamlPath, "STATS.") {
		sensor := parseSensorFromPath(yamlPath)
		wType := parseWidgetTypeFromPath(yamlPath)
		setSelect(p.sensorSelect, sensor)
		setSelect(p.typeSelect, wType)
		p.sensorSelect.Show()
		p.typeSelect.Show()
	} else {
		p.sensorSelect.Hide()
		p.typeSelect.Hide()
	}

	var typePanel fyne.CanvasObject
	var scrollType string
	switch v := obj.(type) {
	case *DraggableWidget:
		setLabel(p.headerLabel, "Propriedades: "+v.YAMLPath)
		setLabel(p.yamlPathLabel, v.YAMLPath)
		setEntry(p.xEntry, strconv.Itoa(v.TextData.X))
		setEntry(p.yEntry, strconv.Itoa(v.TextData.Y))
		if strings.HasPrefix(v.YAMLPath, "static_texts.") {
			setEntry(p.textContentEntry, v.TextData.Text)
		} else {
			setEntry(p.textContentEntry, v.TextData.Placeholder)
		}
		setSelect(p.fontSelector, v.TextData.Font)
		setEntry(p.fontSizeEntry, strconv.Itoa(v.TextData.FontSize))
		setEntry(p.fontColorEntry, v.TextData.FontColor)
		setEntry(p.bgColorEntry, v.TextData.BackgroundColor)
		setSelect(p.alignSelect, strings.ToUpper(v.TextData.Align))
		setSelect(p.formatSelect, strings.ToUpper(v.TextData.Format))
		setCheck(p.showUnitCheck, v.TextData.ShowUnit)
		typePanel = p.textFields
		scrollType = "text"
	case *DraggableGraph:
		setLabel(p.headerLabel, "Propriedades: "+v.YAMLPath)
		setLabel(p.yamlPathLabel, v.YAMLPath)
		setEntry(p.xEntry, strconv.Itoa(v.GraphData.X))
		setEntry(p.yEntry, strconv.Itoa(v.GraphData.Y))
		setEntry(p.graphWidthEntry, strconv.Itoa(v.GraphData.Width))
		setEntry(p.graphHeightEntry, strconv.Itoa(v.GraphData.Height))
		setEntry(p.graphMinEntry, strconv.Itoa(v.GraphData.MinValue))
		setEntry(p.graphMaxEntry, strconv.Itoa(v.GraphData.MaxValue))
		setSelect(p.graphDirectionSelect, v.GraphData.Direction)
		setEntry(p.graphBarColorEntry, v.GraphData.BarColor)
		setEntry(p.graphBgColorEntry, v.GraphData.BackgroundColor)
		setEntry(p.graphGradientColorEntry, v.GraphData.GradientColor)
		setCheck(p.graphOutlineCheck, v.GraphData.BarOutline)
		setCheck(p.graphRevertValueCheck, v.GraphData.RevertValue)
		setEntry(p.graphStepsEntry, strconv.Itoa(v.GraphData.Steps))
		setEntry(p.graphStepGapEntry, strconv.Itoa(v.GraphData.StepGap))
		setEntry(p.graphBlockWidthEntry, strconv.Itoa(v.GraphData.BlockWidth))
		setEntry(p.graphCornerRadiusEntry, strconv.Itoa(v.GraphData.CornerRadius))
		setEntry(p.graphBorderWidthEntry, strconv.Itoa(v.GraphData.BorderWidth))
		typePanel = p.graphFields
		scrollType = "graph"
	case *DraggableRadial:
		setLabel(p.headerLabel, "Propriedades: "+v.YAMLPath)
		setLabel(p.yamlPathLabel, v.YAMLPath)
		setEntry(p.xEntry, strconv.Itoa(v.RadialData.X))
		setEntry(p.yEntry, strconv.Itoa(v.RadialData.Y))
		setEntry(p.radialRadiusEntry, strconv.Itoa(v.RadialData.Radius))
		setEntry(p.radialWidthEntry, strconv.Itoa(v.RadialData.Width))
		setEntry(p.radialMinEntry, strconv.Itoa(v.RadialData.MinValue))
		setEntry(p.radialMaxEntry, strconv.Itoa(v.RadialData.MaxValue))
		setEntry(p.radialStartEntry, strconv.Itoa(v.RadialData.AngleStart))
		setEntry(p.radialEndEntry, strconv.Itoa(v.RadialData.AngleEnd))
		setEntry(p.radialStepsEntry, strconv.Itoa(v.RadialData.AngleSteps))
		setEntry(p.radialSepEntry, strconv.Itoa(v.RadialData.AngleSep))
		setEntry(p.radialBlockAngleEntry, strconv.Itoa(v.RadialData.BlockAngle))
		setCheck(p.radialClockCheck, v.RadialData.Clockwise)
		setCheck(p.radialRoundCheck, v.RadialData.Round)
		setCheck(p.radialRevertCheck, v.RadialData.Revert)
		setCheck(p.radialRevertValueCheck, v.RadialData.RevertValue)
		setEntry(p.radialBarColorEntry, v.RadialData.BarColor)
		setEntry(p.radialBgColorEntry, v.RadialData.BackgroundColor)
		setEntry(p.radialGradientColorEntry, v.RadialData.GradientColor)
		setCheck(p.radialShowTextCheck, v.RadialData.ShowText)
		setCheck(p.radialShowUnitCheck, v.RadialData.ShowUnit)
		setSelect(p.radialFontSelector, v.RadialData.Font)
		setEntry(p.radialFontColorEntry, v.RadialData.FontColor)
		typePanel = p.radialFields
		scrollType = "radial"
	case *DraggableChart:
		setLabel(p.headerLabel, "Propriedades: "+v.YAMLPath)
		setLabel(p.yamlPathLabel, v.YAMLPath)
		setEntry(p.xEntry, strconv.Itoa(v.ChartData.X))
		setEntry(p.yEntry, strconv.Itoa(v.ChartData.Y))
		setEntry(p.chartWidthEntry, strconv.Itoa(v.ChartData.Width))
		setEntry(p.chartHeightEntry, strconv.Itoa(v.ChartData.Height))
		setEntry(p.chartMinEntry, strconv.Itoa(v.ChartData.MinValue))
		setEntry(p.chartMaxEntry, strconv.Itoa(v.ChartData.MaxValue))
		setEntry(p.chartColWidthEntry, strconv.Itoa(v.ChartData.ColumnWidth))
		setEntry(p.chartColGapEntry, strconv.Itoa(v.ChartData.ColumnGap))
		setEntry(p.chartFillColorEntry, v.ChartData.FillColor)
		setEntry(p.chartLineColorEntry, v.ChartData.LineColor)
		setEntry(p.chartBorderEntry, strconv.Itoa(v.ChartData.BorderWidth))
		typePanel = p.chartFields
		scrollType = "chart"
	}

	if scrollType != p.lastScrollType {
		p.lastScrollType = scrollType
		p.scroll.Content = container.NewVBox(p.commonFields, typePanel)
		p.scroll.Refresh()
	}
}

// onSensorChanged handles the Sensor select change.
// Moves the current widget to a different sensor path, preserving all data.
func (p *PropertiesPanel) onSensorChanged(newSensor string) {
	if p.selectedWidget == nil {
		return
	}

	var currentPath string
	var widgetType string

	switch v := p.selectedWidget.(type) {
	case *DraggableWidget:
		currentPath = v.YAMLPath
		widgetType = parseWidgetTypeFromPath(currentPath)
	case *DraggableGraph:
		currentPath = v.YAMLPath
		widgetType = "GRAPH"
	case *DraggableRadial:
		currentPath = v.YAMLPath
		widgetType = "RADIAL"
	case *DraggableChart:
		currentPath = v.YAMLPath
		widgetType = "CHART"
	default:
		return
	}

	currentSensor := parseSensorFromPath(currentPath)
	if currentSensor == newSensor {
		return
	}

	oldMeasurement := getMeasurementForSensor(p.app.currentTheme, currentSensor)
	newMeasurement := getMeasurementForSensor(p.app.currentTheme, newSensor)

	if newMeasurement != nil {
		clearWidgetType(newMeasurement, widgetType)
		switch v := p.selectedWidget.(type) {
		case *DraggableWidget:
			cloned := *v.TextData
			switch widgetType {
			case "TEXT":
				newMeasurement.Text = &cloned
			case "PERCENT_TEXT":
				newMeasurement.PercentText = &cloned
			}
		case *DraggableGraph:
			cloned := *v.GraphData
			newMeasurement.Graph = &cloned
		case *DraggableRadial:
			cloned := *v.RadialData
			newMeasurement.Radial = &cloned
		case *DraggableChart:
			cloned := *v.ChartData
			newMeasurement.Chart = &cloned
		}
	}

	if oldMeasurement != nil {
		clearWidgetType(oldMeasurement, widgetType)
	}

	newPath := "STATS." + newSensor + "." + widgetType
	p.app.RefreshCanvas()
	p.app.layersPanel.Refresh()
	p.app.selectWidgetByPath(newPath)
}

// onTypeChanged handles the Type select change.
// Changes the visual representation of the currently selected sensor, preserving compatible fields.
func (p *PropertiesPanel) onTypeChanged(newType string) {
	if p.selectedWidget == nil {
		return
	}

	var currentPath string
	var snap widgetSnapshot

	switch v := p.selectedWidget.(type) {
	case *DraggableWidget:
		currentPath = v.YAMLPath
		snap = widgetSnapshot{
			X: v.TextData.X, Y: v.TextData.Y,
			Font: v.TextData.Font, FontSize: v.TextData.FontSize,
			PrimaryColor:    v.TextData.FontColor,
			FontColor:       v.TextData.FontColor,
			BackgroundColor: v.TextData.BackgroundColor,
			Width: 150, Height: 15,
			Radius: 40, ArcWidth: 10,
			AngleStart: 36, AngleEnd: 60, AngleSteps: 1,
		}
	case *DraggableGraph:
		currentPath = v.YAMLPath
		snap = widgetSnapshot{
			X: v.GraphData.X, Y: v.GraphData.Y,
			Index:           v.GraphData.Index,
			Width:           v.GraphData.Width,
			Height:          v.GraphData.Height,
			MinValue:        v.GraphData.MinValue,
			MaxValue:        v.GraphData.MaxValue,
			PrimaryColor:    v.GraphData.BarColor,
			BackgroundColor: v.GraphData.BackgroundColor,
			GradientColor:   v.GraphData.GradientColor,
			Radius: 40, ArcWidth: 10,
			AngleStart: 36, AngleEnd: 60, AngleSteps: 1,
		}
	case *DraggableRadial:
		currentPath = v.YAMLPath
		snap = widgetSnapshot{
			X: v.RadialData.X, Y: v.RadialData.Y,
			Index:           v.RadialData.Index,
			Radius:          v.RadialData.Radius,
			ArcWidth:        v.RadialData.Width,
			MinValue:        v.RadialData.MinValue,
			MaxValue:        v.RadialData.MaxValue,
			AngleStart:      v.RadialData.AngleStart,
			AngleEnd:        v.RadialData.AngleEnd,
			AngleSteps:      v.RadialData.AngleSteps,
			AngleSep:        v.RadialData.AngleSep,
			BlockAngle:      v.RadialData.BlockAngle,
			Clockwise:       v.RadialData.Clockwise,
			Round:           v.RadialData.Round,
			PrimaryColor:    v.RadialData.BarColor,
			BackgroundColor: v.RadialData.BackgroundColor,
			GradientColor:   v.RadialData.GradientColor,
			Font:            v.RadialData.Font,
			FontSize:        22,
			FontColor:       v.RadialData.FontColor,
			Width:           v.RadialData.Radius * 2,
			Height:          15,
		}
	case *DraggableChart:
		currentPath = v.YAMLPath
		snap = widgetSnapshot{
			X: v.ChartData.X, Y: v.ChartData.Y,
			Index:        v.ChartData.Index,
			Width:        v.ChartData.Width,
			Height:       v.ChartData.Height,
			MinValue:     v.ChartData.MinValue,
			MaxValue:     v.ChartData.MaxValue,
			PrimaryColor: v.ChartData.FillColor,
			Radius: 40, ArcWidth: 10,
			AngleStart: 36, AngleEnd: 60, AngleSteps: 1,
		}
	default:
		return
	}

	currentType := parseWidgetTypeFromPath(currentPath)
	if currentType == newType {
		return
	}

	sensor := parseSensorFromPath(currentPath)
	m := getMeasurementForSensor(p.app.currentTheme, sensor)
	if m != nil {
		setWidgetOnMeasurement(m, newType, snap)
	}

	// Refresh and select the widget at the new path
	newPath := "STATS." + sensor + "." + newType
	p.app.RefreshCanvas()
	p.app.layersPanel.Refresh()
	p.app.selectWidgetByPath(newPath)
}

// clearWidgetType removes a specific widget type from a measurement.
func clearWidgetType(m *theme.Measurement, widgetType string) {
	switch widgetType {
	case "TEXT":
		m.Text = nil
	case "PERCENT_TEXT":
		m.PercentText = nil
	case "GRAPH":
		m.Graph = nil
	case "RADIAL":
		m.Radial = nil
	case "CHART":
		m.Chart = nil
	case "STATUS_BAR":
		m.StatusBar = nil
	case "GAUGE":
		m.Gauge = nil
	}
}

// ShowLayerInfo shows properties for a background layer or video.
func (p *PropertiesPanel) ShowLayerInfo(name, path string, onChange func()) {
	p.selectedWidget = nil
	p.sensorSelect.Hide()
	p.typeSelect.Hide()

	setLabel(p.headerLabel, "Layer: "+name)

	changeBtn := widget.NewButton("Trocar Arquivo...", func() {
		onChange()
	})
	layerContent := container.NewVBox(
		widget.NewLabel("Arquivo:"),
		widget.NewLabel(path),
		widget.NewSeparator(),
		changeBtn,
	)
	p.scroll.Content = layerContent
	p.scroll.Refresh()
}
