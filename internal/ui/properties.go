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

	refreshMu    sync.Mutex
	refreshTimer *time.Timer

	sensorSelect  *widget.Select
	typeSelect    *widget.Select
	textFields    *fyne.Container
	graphFields   *fyne.Container
	radialFields  *fyne.Container
	chartFields   *fyne.Container
	commonFields  *fyne.Container
	placeholder   *widget.Label
	scroll        *container.Scroll
	scrollContent *fyne.Container

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

// scheduleRefresh debounces visual refreshes so that rapid OnChanged events
// (e.g. every keystroke in a text entry) only trigger one buildImage() call
// after 150ms of inactivity instead of one per keystroke.
func (p *PropertiesPanel) scheduleRefresh(w fyne.Widget) {
	p.refreshMu.Lock()
	defer p.refreshMu.Unlock()
	if p.refreshTimer != nil {
		p.refreshTimer.Stop()
	}
	p.refreshTimer = time.AfterFunc(150*time.Millisecond, func() {
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
			}, p.app.activeWindow())
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
			}, p.app.activeWindow())
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
	p.textFields = container.NewVBox(
		widget.NewLabel("Texto:"),
		p.textContentEntry,
		widget.NewLabel("Fonte:"),
		p.fontSelector,
		widget.NewLabel("Tamanho:"),
		p.fontSizeEntry,
		widget.NewLabel("Cor da Fonte (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.fontColorBtn, p.fontColorEntry),
		widget.NewLabel("Cor de Fundo (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.bgColorBtn, p.bgColorEntry),
		widget.NewLabel("Alinhamento:"),
		p.alignSelect,
		widget.NewLabel("Formato (datas):"),
		p.formatSelect,
		p.showUnitCheck,
	)

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
			}, p.app.activeWindow())
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
			}, p.app.activeWindow())
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
			}, p.app.activeWindow())
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
	p.graphFields = container.NewVBox(
		widget.NewLabel("Largura:"), p.graphWidthEntry,
		widget.NewLabel("Altura:"), p.graphHeightEntry,
		widget.NewLabel("Valor Mínimo:"), p.graphMinEntry,
		widget.NewLabel("Valor Máximo:"), p.graphMaxEntry,
		widget.NewLabel("Direção:"), p.graphDirectionSelect,
		widget.NewLabel("Cor da Barra (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.graphBarColorBtn, p.graphBarColorEntry),
		widget.NewLabel("Cor de Fundo (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.graphBgColorBtn, p.graphBgColorEntry),
		widget.NewLabel("Gradiente (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.graphGradientColorBtn, p.graphGradientColorEntry),
		p.graphOutlineCheck,
		p.graphRevertValueCheck,
		widget.NewLabel("Segmentos / Gap:"),
		container.NewGridWithColumns(2, p.graphStepsEntry, p.graphStepGapEntry),
		widget.NewLabel("Largura do Bloco:"), p.graphBlockWidthEntry,
		widget.NewLabel("Raio do Canto:"), p.graphCornerRadiusEntry,
		widget.NewLabel("Espessura da Borda:"), p.graphBorderWidthEntry,
	)

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
			}, p.app.activeWindow())
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
			}, p.app.activeWindow())
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
			}, p.app.activeWindow())
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
			}, p.app.activeWindow())
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
	p.radialFields = container.NewVBox(
		widget.NewLabel("Raio:"), p.radialRadiusEntry,
		widget.NewLabel("Espessura:"), p.radialWidthEntry,
		widget.NewLabel("Valor Mín/Máx:"),
		container.NewGridWithColumns(2, p.radialMinEntry, p.radialMaxEntry),
		widget.NewLabel("Ângulo Início/Fim:"),
		container.NewGridWithColumns(2, p.radialStartEntry, p.radialEndEntry),
		widget.NewLabel("Segmentos / Separação:"),
		container.NewGridWithColumns(2, p.radialStepsEntry, p.radialSepEntry),
		widget.NewLabel("Ângulo do Bloco (graus):"), p.radialBlockAngleEntry,
		p.radialClockCheck,
		p.radialRoundCheck,
		p.radialRevertCheck,
		p.radialRevertValueCheck,
		widget.NewLabel("Cor do Arco (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.radialBarColorBtn, p.radialBarColorEntry),
		widget.NewLabel("Cor de Fundo (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.radialBgColorBtn, p.radialBgColorEntry),
		widget.NewLabel("Gradiente (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.radialGradientColorBtn, p.radialGradientColorEntry),
		p.radialShowTextCheck,
		p.radialShowUnitCheck,
		widget.NewLabel("Fonte:"), p.radialFontSelector,
		widget.NewLabel("Cor da Fonte (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.radialFontColorBtn, p.radialFontColorEntry),
	)

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
			}, p.app.activeWindow())
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
			}, p.app.activeWindow())
			picker.Show()
		}
	})
	p.chartBorderEntry = makeIntEntry(func(v int) {
		if dc, ok := p.selectedWidget.(*DraggableChart); ok {
			dc.ChartData.BorderWidth = v
			p.scheduleRefresh(dc)
		}
	})
	p.chartFields = container.NewVBox(
		widget.NewLabel("Largura:"), p.chartWidthEntry,
		widget.NewLabel("Altura:"), p.chartHeightEntry,
		widget.NewLabel("Valor Mín/Máx:"),
		container.NewGridWithColumns(2, p.chartMinEntry, p.chartMaxEntry),
		widget.NewLabel("Largura Coluna:"), p.chartColWidthEntry,
		widget.NewLabel("Gap Coluna:"), p.chartColGapEntry,
		widget.NewLabel("Cor Preenchimento (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.chartFillColorBtn, p.chartFillColorEntry),
		widget.NewLabel("Cor da Linha (R,G,B):"),
		container.NewBorder(nil, nil, nil, p.chartLineColorBtn, p.chartLineColorEntry),
		widget.NewLabel("Borda:"), p.chartBorderEntry,
	)

	// === PLACEHOLDER ===
	p.placeholder = widget.NewLabel("Selecione um componente para editar suas propriedades.")
	p.placeholder.Wrapping = fyne.TextWrapWord
	p.placeholder.Alignment = fyne.TextAlignCenter

	p.scrollContent = container.NewVBox(
		p.commonFields,
		p.textFields,
		p.graphFields,
		p.radialFields,
		p.chartFields,
	)
	p.scroll = container.NewVScroll(p.scrollContent)
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
	p.textFields.Hide()
	p.graphFields.Hide()
	p.radialFields.Hide()
	p.chartFields.Hide()
	p.commonFields.Hide()

	if obj == nil {
		log.Printf("[PropertiesPanel] Update(nil) — mostrando placeholder")
		p.headerLabel.SetText("Propriedades")
		p.scroll.Content = p.placeholder
		p.scroll.Refresh()
		return
	}

	log.Printf("[PropertiesPanel] Update(%T) — mostrando propriedades", obj)
	p.scroll.Content = p.scrollContent
	p.commonFields.Show()

	// Populate sensor/type selects based on widget path
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
		p.sensorSelect.SetSelected(sensor)
		p.typeSelect.SetSelected(wType)
		p.sensorSelect.Show()
		p.typeSelect.Show()
	} else {
		p.sensorSelect.Hide()
		p.typeSelect.Hide()
	}

	switch v := obj.(type) {
	case *DraggableWidget:
		p.headerLabel.SetText("Propriedades: " + v.YAMLPath)
		p.textFields.Show()
		p.yamlPathLabel.SetText(v.YAMLPath)
		p.xEntry.SetText(strconv.Itoa(v.TextData.X))
		p.yEntry.SetText(strconv.Itoa(v.TextData.Y))
		if strings.HasPrefix(v.YAMLPath, "static_texts.") {
			p.textContentEntry.SetText(v.TextData.Text)
		} else {
			p.textContentEntry.SetText(v.TextData.Placeholder)
		}
		p.fontSelector.SetSelected(v.TextData.Font)
		p.fontSizeEntry.SetText(strconv.Itoa(v.TextData.FontSize))
		p.fontColorEntry.SetText(v.TextData.FontColor)
		p.bgColorEntry.SetText(v.TextData.BackgroundColor)
		p.alignSelect.SetSelected(strings.ToUpper(v.TextData.Align))
		p.formatSelect.SetSelected(strings.ToUpper(v.TextData.Format))
		p.showUnitCheck.SetChecked(v.TextData.ShowUnit)
	case *DraggableGraph:
		p.headerLabel.SetText("Propriedades: " + v.YAMLPath)
		p.graphFields.Show()
		p.yamlPathLabel.SetText(v.YAMLPath)
		p.xEntry.SetText(strconv.Itoa(v.GraphData.X))
		p.yEntry.SetText(strconv.Itoa(v.GraphData.Y))
		p.graphWidthEntry.SetText(strconv.Itoa(v.GraphData.Width))
		p.graphHeightEntry.SetText(strconv.Itoa(v.GraphData.Height))
		p.graphMinEntry.SetText(strconv.Itoa(v.GraphData.MinValue))
		p.graphMaxEntry.SetText(strconv.Itoa(v.GraphData.MaxValue))
		p.graphDirectionSelect.SetSelected(v.GraphData.Direction)
		p.graphBarColorEntry.SetText(v.GraphData.BarColor)
		p.graphBgColorEntry.SetText(v.GraphData.BackgroundColor)
		p.graphGradientColorEntry.SetText(v.GraphData.GradientColor)
		p.graphOutlineCheck.SetChecked(v.GraphData.BarOutline)
		p.graphRevertValueCheck.SetChecked(v.GraphData.RevertValue)
		p.graphStepsEntry.SetText(strconv.Itoa(v.GraphData.Steps))
		p.graphStepGapEntry.SetText(strconv.Itoa(v.GraphData.StepGap))
		p.graphBlockWidthEntry.SetText(strconv.Itoa(v.GraphData.BlockWidth))
		p.graphCornerRadiusEntry.SetText(strconv.Itoa(v.GraphData.CornerRadius))
		p.graphBorderWidthEntry.SetText(strconv.Itoa(v.GraphData.BorderWidth))
	case *DraggableRadial:
		p.headerLabel.SetText("Propriedades: " + v.YAMLPath)
		p.radialFields.Show()
		p.yamlPathLabel.SetText(v.YAMLPath)
		p.xEntry.SetText(strconv.Itoa(v.RadialData.X))
		p.yEntry.SetText(strconv.Itoa(v.RadialData.Y))
		p.radialRadiusEntry.SetText(strconv.Itoa(v.RadialData.Radius))
		p.radialWidthEntry.SetText(strconv.Itoa(v.RadialData.Width))
		p.radialMinEntry.SetText(strconv.Itoa(v.RadialData.MinValue))
		p.radialMaxEntry.SetText(strconv.Itoa(v.RadialData.MaxValue))
		p.radialStartEntry.SetText(strconv.Itoa(v.RadialData.AngleStart))
		p.radialEndEntry.SetText(strconv.Itoa(v.RadialData.AngleEnd))
		p.radialStepsEntry.SetText(strconv.Itoa(v.RadialData.AngleSteps))
		p.radialSepEntry.SetText(strconv.Itoa(v.RadialData.AngleSep))
		p.radialBlockAngleEntry.SetText(strconv.Itoa(v.RadialData.BlockAngle))
		p.radialClockCheck.SetChecked(v.RadialData.Clockwise)
		p.radialRoundCheck.SetChecked(v.RadialData.Round)
		p.radialRevertCheck.SetChecked(v.RadialData.Revert)
		p.radialRevertValueCheck.SetChecked(v.RadialData.RevertValue)
		p.radialBarColorEntry.SetText(v.RadialData.BarColor)
		p.radialBgColorEntry.SetText(v.RadialData.BackgroundColor)
		p.radialGradientColorEntry.SetText(v.RadialData.GradientColor)
		p.radialShowTextCheck.SetChecked(v.RadialData.ShowText)
		p.radialShowUnitCheck.SetChecked(v.RadialData.ShowUnit)
		p.radialFontSelector.SetSelected(v.RadialData.Font)
		p.radialFontColorEntry.SetText(v.RadialData.FontColor)
	case *DraggableChart:
		p.headerLabel.SetText("Propriedades: " + v.YAMLPath)
		p.chartFields.Show()
		p.yamlPathLabel.SetText(v.YAMLPath)
		p.xEntry.SetText(strconv.Itoa(v.ChartData.X))
		p.yEntry.SetText(strconv.Itoa(v.ChartData.Y))
		p.chartWidthEntry.SetText(strconv.Itoa(v.ChartData.Width))
		p.chartHeightEntry.SetText(strconv.Itoa(v.ChartData.Height))
		p.chartMinEntry.SetText(strconv.Itoa(v.ChartData.MinValue))
		p.chartMaxEntry.SetText(strconv.Itoa(v.ChartData.MaxValue))
		p.chartColWidthEntry.SetText(strconv.Itoa(v.ChartData.ColumnWidth))
		p.chartColGapEntry.SetText(strconv.Itoa(v.ChartData.ColumnGap))
		p.chartFillColorEntry.SetText(v.ChartData.FillColor)
		p.chartLineColorEntry.SetText(v.ChartData.LineColor)
		p.chartBorderEntry.SetText(strconv.Itoa(v.ChartData.BorderWidth))
	}

	p.scroll.Refresh()
}

// onSensorChanged handles the Sensor select change.
// Moves the current widget to a different sensor path in the theme.
func (p *PropertiesPanel) onSensorChanged(newSensor string) {
	if p.selectedWidget == nil {
		return
	}

	var currentPath string
	var x, y int
	var font string
	var fontSize int
	var fontColor string

	switch v := p.selectedWidget.(type) {
	case *DraggableWidget:
		currentPath = v.YAMLPath
		x, y = v.TextData.X, v.TextData.Y
		font = v.TextData.Font
		fontSize = v.TextData.FontSize
		fontColor = v.TextData.FontColor
	case *DraggableGraph:
		currentPath = v.YAMLPath
		x, y = v.GraphData.X, v.GraphData.Y
		fontColor = v.GraphData.BarColor
	case *DraggableRadial:
		currentPath = v.YAMLPath
		x, y = v.RadialData.X, v.RadialData.Y
		fontColor = v.RadialData.BarColor
	case *DraggableChart:
		currentPath = v.YAMLPath
		x, y = v.ChartData.X, v.ChartData.Y
		fontColor = v.ChartData.FillColor
	default:
		return
	}

	currentSensor := parseSensorFromPath(currentPath)
	if currentSensor == newSensor {
		return // same sensor, no change
	}

	widgetType := parseWidgetTypeFromPath(currentPath)
	if font == "" {
		font = "jetbrains-mono/JetBrainsMono-Bold.ttf"
	}
	if fontSize == 0 {
		fontSize = 22
	}
	if fontColor == "" {
		fontColor = "255,255,255"
	}

	// Remove from old sensor
	oldMeasurement := getMeasurementForSensor(p.app.currentTheme, currentSensor)
	if oldMeasurement != nil {
		clearWidgetType(oldMeasurement, widgetType)
	}

	// Add to new sensor
	newMeasurement := getMeasurementForSensor(p.app.currentTheme, newSensor)
	if newMeasurement != nil {
		setWidgetOnMeasurement(newMeasurement, widgetType, x, y, font, fontSize, fontColor)
	}

	// Refresh and select the widget at the new path
	newPath := "STATS." + newSensor + "." + widgetType
	p.app.RefreshCanvas()
	p.app.layersPanel.Refresh()
	p.app.selectWidgetByPath(newPath)
}

// onTypeChanged handles the Type select change.
// Changes the visual representation of the currently selected sensor.
func (p *PropertiesPanel) onTypeChanged(newType string) {
	if p.selectedWidget == nil {
		return
	}

	var currentPath string
	var x, y int
	var font string
	var fontSize int
	var fontColor string

	switch v := p.selectedWidget.(type) {
	case *DraggableWidget:
		currentPath = v.YAMLPath
		x, y = v.TextData.X, v.TextData.Y
		font = v.TextData.Font
		fontSize = v.TextData.FontSize
		fontColor = v.TextData.FontColor
	case *DraggableGraph:
		currentPath = v.YAMLPath
		x, y = v.GraphData.X, v.GraphData.Y
		fontColor = v.GraphData.BarColor
	case *DraggableRadial:
		currentPath = v.YAMLPath
		x, y = v.RadialData.X, v.RadialData.Y
		fontColor = v.RadialData.BarColor
	case *DraggableChart:
		currentPath = v.YAMLPath
		x, y = v.ChartData.X, v.ChartData.Y
		fontColor = v.ChartData.FillColor
	default:
		return
	}

	currentType := parseWidgetTypeFromPath(currentPath)
	if currentType == newType {
		return
	}

	sensor := parseSensorFromPath(currentPath)
	if font == "" {
		font = "jetbrains-mono/JetBrainsMono-Bold.ttf"
	}
	if fontSize == 0 {
		fontSize = 22
	}
	if fontColor == "" {
		fontColor = "255,255,255"
	}

	// Replace widget type on the same measurement
	m := getMeasurementForSensor(p.app.currentTheme, sensor)
	if m != nil {
		setWidgetOnMeasurement(m, newType, x, y, font, fontSize, fontColor)
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
	p.textFields.Hide()
	p.graphFields.Hide()
	p.radialFields.Hide()
	p.chartFields.Hide()
	p.sensorSelect.Hide()
	p.typeSelect.Hide()

	p.headerLabel.SetText("Layer: " + name)
	p.yamlPathLabel.SetText(path)
	p.xEntry.SetText("")
	p.yEntry.SetText("")

	p.commonFields.Show()
	p.deleteButton.Hide()

	// Replace scroll content with layer-specific UI
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
