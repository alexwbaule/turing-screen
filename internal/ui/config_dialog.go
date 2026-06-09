package ui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/spf13/viper"
)

func ShowConfigDialog(app *EditorApp) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile("conf/config.yaml")
	if err := v.ReadInConfig(); err != nil {
		dialog.ShowError(fmt.Errorf("erro ao ler config: %w", err), app.activeWindow())
		return
	}

	// --- Geral ---
	portEntry := widget.NewEntry()
	portEntry.SetText(v.GetString("device.port"))
	portEntry.SetPlaceHolder("AUTO")

	apiPortEntry := widget.NewEntry()
	apiPortEntry.SetText(strconv.Itoa(v.GetInt("device.api_port")))

	logSelect := widget.NewSelect([]string{"debug", "info", "warn", "error"}, nil)
	logSelect.SetSelected(v.GetString("device.log"))

	turnOffCheck := widget.NewCheck("Desligar display ao sair", nil)
	turnOffCheck.SetChecked(v.GetBool("device.turn_off_on_exit"))

	generalForm := widget.NewForm(
		widget.NewFormItem("Porta Serial", portEntry),
		widget.NewFormItem("API Port", apiPortEntry),
		widget.NewFormItem("Log Level", logSelect),
		widget.NewFormItem("", turnOffCheck),
	)

	// --- Display ---
	brightnessSlider := widget.NewSlider(0, 100)
	brightnessSlider.SetValue(float64(v.GetInt("device.display.brightness")))
	brightnessLabel := widget.NewLabel(fmt.Sprintf("%d%%", int(brightnessSlider.Value)))
	brightnessSlider.OnChanged = func(val float64) {
		brightnessLabel.SetText(fmt.Sprintf("%d%%", int(val)))
	}

	widthEntry := widget.NewEntry()
	widthEntry.SetText(strconv.Itoa(v.GetInt("device.display.width")))

	heightEntry := widget.NewEntry()
	heightEntry.SetText(strconv.Itoa(v.GetInt("device.display.height")))

	reverseCheck := widget.NewCheck("Inverter orientação", nil)
	reverseCheck.SetChecked(v.GetBool("device.display.reverse"))

	displayForm := widget.NewForm(
		widget.NewFormItem("Brilho", container.NewBorder(nil, nil, nil, brightnessLabel, brightnessSlider)),
		widget.NewFormItem("Largura (px)", widthEntry),
		widget.NewFormItem("Altura (px)", heightEntry),
		widget.NewFormItem("", reverseCheck),
	)

	// --- Sensores ---
	cpuIntervalEntry := widget.NewEntry()
	cpuIntervalEntry.SetText(v.GetString("device.sensors.cpu.interval"))
	cpuIntervalEntry.SetPlaceHolder("1s")

	cpuTempEntry := widget.NewEntry()
	cpuTempEntry.SetText(v.GetString("device.sensors.cpu.temperature_sensor"))
	cpuTempEntry.SetPlaceHolder("auto")

	gpuIntervalEntry := widget.NewEntry()
	gpuIntervalEntry.SetText(v.GetString("device.sensors.gpu.interval"))
	gpuIntervalEntry.SetPlaceHolder("1s")

	gpuProviderSelect := widget.NewSelect([]string{"auto", "nvidia", "amd", "noop"}, nil)
	gpuProviderSelect.SetSelected(v.GetString("device.sensors.gpu.provider"))

	memIntervalEntry := widget.NewEntry()
	memIntervalEntry.SetText(v.GetString("device.sensors.memory.interval"))
	memIntervalEntry.SetPlaceHolder("5s")

	diskIntervalEntry := widget.NewEntry()
	diskIntervalEntry.SetText(v.GetString("device.sensors.disk.interval"))
	diskIntervalEntry.SetPlaceHolder("10s")

	diskTempEntry := widget.NewEntry()
	diskTempEntry.SetText(v.GetString("device.sensors.disk.temperature_sensor"))
	diskTempEntry.SetPlaceHolder("auto")

	ethEntry := widget.NewEntry()
	ethEntry.SetText(v.GetString("device.sensors.network.eth"))
	ethEntry.SetPlaceHolder("enp3s0")

	wifiEntry := widget.NewEntry()
	wifiEntry.SetText(v.GetString("device.sensors.network.wlo"))
	wifiEntry.SetPlaceHolder("wlp4s0")

	netIntervalEntry := widget.NewEntry()
	netIntervalEntry.SetText(v.GetString("device.sensors.network.interval"))
	netIntervalEntry.SetPlaceHolder("1s")

	sensorsForm := container.NewVBox(
		widget.NewLabelWithStyle("CPU", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Intervalo", cpuIntervalEntry),
			widget.NewFormItem("Sensor de Temperatura", cpuTempEntry),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("GPU", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Intervalo", gpuIntervalEntry),
			widget.NewFormItem("Provider", gpuProviderSelect),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Memória", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Intervalo", memIntervalEntry),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Disco", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Intervalo", diskIntervalEntry),
			widget.NewFormItem("Sensor de Temperatura", diskTempEntry),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Rede", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Interface Cabeada", ethEntry),
			widget.NewFormItem("Interface Wi-Fi", wifiEntry),
			widget.NewFormItem("Intervalo", netIntervalEntry),
		),
	)

	// --- Weather ---
	weatherEnabledCheck := widget.NewCheck("Habilitar clima", nil)
	weatherEnabledCheck.SetChecked(v.GetBool("device.sensors.weather.enabled"))

	weatherCityEntry := widget.NewEntry()
	weatherCityEntry.SetText(v.GetString("device.sensors.weather.city"))
	weatherCityEntry.SetPlaceHolder("São Paulo,BR")

	weatherIntervalEntry := widget.NewEntry()
	weatherIntervalEntry.SetText(v.GetString("device.sensors.weather.interval"))
	weatherIntervalEntry.SetPlaceHolder("30m")

	weatherForm := widget.NewForm(
		widget.NewFormItem("", weatherEnabledCheck),
		widget.NewFormItem("Cidade", weatherCityEntry),
		widget.NewFormItem("Intervalo", weatherIntervalEntry),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Geral", container.NewVScroll(generalForm)),
		container.NewTabItem("Display", container.NewVScroll(displayForm)),
		container.NewTabItem("Sensores", container.NewVScroll(sensorsForm)),
		container.NewTabItem("Clima", container.NewVScroll(weatherForm)),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	content := container.NewPadded(tabs)
	content.Resize(fyne.NewSize(500, 420))

	dlg := dialog.NewCustomConfirm("Configurações", "Salvar", "Cancelar", content,
		func(save bool) {
			if !save {
				return
			}

			apiPort, _ := strconv.Atoi(apiPortEntry.Text)
			width, _ := strconv.Atoi(widthEntry.Text)
			height, _ := strconv.Atoi(heightEntry.Text)

			v.Set("device.port", portEntry.Text)
			v.Set("device.api_port", apiPort)
			v.Set("device.log", logSelect.Selected)
			v.Set("device.turn_off_on_exit", turnOffCheck.Checked)

			v.Set("device.display.brightness", int(brightnessSlider.Value))
			v.Set("device.display.width", width)
			v.Set("device.display.height", height)
			v.Set("device.display.reverse", reverseCheck.Checked)

			v.Set("device.sensors.cpu.interval", cpuIntervalEntry.Text)
			v.Set("device.sensors.cpu.temperature_sensor", cpuTempEntry.Text)
			v.Set("device.sensors.gpu.interval", gpuIntervalEntry.Text)
			v.Set("device.sensors.gpu.provider", gpuProviderSelect.Selected)
			v.Set("device.sensors.memory.interval", memIntervalEntry.Text)
			v.Set("device.sensors.disk.interval", diskIntervalEntry.Text)
			v.Set("device.sensors.disk.temperature_sensor", diskTempEntry.Text)
			v.Set("device.sensors.network.eth", ethEntry.Text)
			v.Set("device.sensors.network.wlo", wifiEntry.Text)
			v.Set("device.sensors.network.interval", netIntervalEntry.Text)
			v.Set("device.sensors.weather.enabled", weatherEnabledCheck.Checked)
			v.Set("device.sensors.weather.city", weatherCityEntry.Text)
			v.Set("device.sensors.weather.interval", weatherIntervalEntry.Text)

			if err := v.WriteConfig(); err != nil {
				dialog.ShowError(fmt.Errorf("erro ao salvar config: %w", err), app.activeWindow())
				return
			}
			dialog.ShowInformation("Configurações", "Salvo. Reinicie o daemon para aplicar as alterações.", app.activeWindow())
		}, app.activeWindow())

	dlg.Resize(fyne.NewSize(520, 460))
	dlg.Show()
}
