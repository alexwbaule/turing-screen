# TODO — Implementação Go do Turing Smart Screen 5"

## Problema Atual (em investigação)

O vídeo toca mas os sensor updates não aparecem visualmente. O device aceita os updates (`needReSend:0|renderCnt:N`) mas não renderiza.

**Descoberta**: No PCAP, a diferença de posição entre linhas consecutivas de um sensor update é **803** (não 800). Isso sugere que o display width usado no cálculo de posição é 803, não 800. Ou há um padding de 3 bytes por linha.

**Próximo passo**: Investigar se o display width no cálculo de posição precisa ser ajustado para video mode, ou se o formato dos sensor updates é diferente quando vídeo está tocando.

**Arquivos relevantes**: 
- `internal/resource/process/device/main.go` — `GeneratePartialImage()` calcula `position = (x0+h)*display.Width + y0`
- `scripts/capture_sensor_update.py` — extrai updates do PCAP para análise

## Fluxos do Sistema

Existem **2 fluxos** baseados no tema carregado (`theme.yaml`):

### Fluxo 1: Imagem Estática (sem `video_play` no tema)
```
HELLO → STOP_VIDEO → STOP_MEDIA → SET_BRIGHTNESS
PRE_UPDATE_BITMAP (0x86)
SEPARATOR (0x2c × 250)
SEND_BITMAP (0xc8 + BGRA 800×480) → "full_png_sucess"
QUERY_STATUS (0xcf) → "needReSend:0|renderCnt:N"
[sensor updates loop — UPDATE_BITMAP 0xcc]
```
**Status**: Funcionava com `tarm/serial`. Precisa validar com `go.bug.st/serial`.

---

### Fluxo 2: Vídeo (com `video_play` no tema)

#### PRÉ-FLUXO (init + upload condicional):
```
HELLO → STOP_VIDEO → STOP_MEDIA → SET_BRIGHTNESS → CMD_0x7D
GET_STORAGE_STATUS
GET_FILE_INFO "/root/video/X.mp4"
```

**SE** `GET_FILE_INFO` retorna 0 (arquivo não existe) → **UPLOAD**:
```
STOP_VIDEO → STOP_MEDIA
LIST_DIR "/root/video/"
CREATE_FILE "/root/video/X.mp4" size=N → "create_success"
[raw file data — single write, DTR=1 RTS=1 obrigatório]
← "file_rev_doneimg_show_" (polling 1s, esperar até 60s)
GET_FILE_INFO → verificar tamanho
```

**SE** `GET_FILE_INFO` retorna > 0 (arquivo já existe) → **PULA O UPLOAD**

#### FLUXO PRINCIPAL (sempre, após pré-fluxo):
```
RESTART_DEVICE (0x82)
(sleep 2s)
HELLO
GET_FILE_INFO "/root/video/X.mp4"
PLAY_VIDEO loop=1 → "play_video_success"
PRE_UPDATE_BITMAP (0x86)
SEPARATOR (0x2c × 250)
SET_BRIGHTNESS
SEND_OVERLAY (0xca + BGRA 800×480) → "seq_png_init_sucess"
[sensor updates loop — UPDATE_BITMAP 0xcc]
```

---

## O que precisa ser feito

### 1. Fix do serial driver (`go.bug.st/serial`)

**Arquivo**: `internal/resource/serial/main.go`

**Problema**: Trocamos de `tarm/serial` para `go.bug.st/serial` para ter DTR/RTS. Mas o read pode estar com comportamento diferente (travou no GET_STORAGE_STATUS).

**O que fazer**:
- Verificar se `SetReadTimeout(5s)` funciona igual ao `tarm/serial` (retorna dados parciais ou espera timeout completo)
- Se necessário, usar timeout menor + retry loop no `Read()`
- Testar primeiro o **Fluxo 1** (imagem estática) para confirmar que a lib funciona

### 2. Implementar o Fluxo 2 completo no `main.go`

**Arquivo**: `cmd/turing-screen/main.go`

**Problema atual**: O código enfileira tudo na queue (incluindo upload), mas o upload precisa ser **síncrono** (usa `WriteRaw` + `ReadPoll` diretamente no serial, não pode ir pela queue do worker).

**Solução**: O pré-fluxo (init + upload condicional) deve rodar **ANTES** do worker iniciar, usando o serial diretamente:

```go
// Pseudo-código do main.go para fluxo de vídeo:

// 1. Abrir serial (DTR=1, RTS=1 já setados)
// 2. Enviar init diretamente (sem worker):
devSerial.Write(cmdDevice.Hello())
devSerial.Write(cmdMedia.StopVideo())
devSerial.Write(cmdMedia.StopMedia())
devSerial.Write(cmdBright.SetBrightness(...))
devSerial.Write(cmdStorage.SetPreUpload(brightness))
devSerial.Write(cmdStorage.GetStorageStatus())  // ler resposta
devSerial.Write(cmdStorage.GetFileInfo(path))   // ler resposta

// 3. Se arquivo não existe → upload:
if fileSize == 0 {
    devSerial.Write(cmdMedia.StopVideo())
    devSerial.Write(cmdMedia.StopMedia())
    devSerial.Write(cmdStorage.CreateFile(path, size))  // ler "create_success"
    devSerial.WriteRaw(fileData)                         // write único
    devSerial.ReadPoll(60 * time.Second)                 // esperar "file_rev_done"
}

// 4. Enfileirar o fluxo principal na queue:
queue.Enqueue(cmdStorage.RestartDevice())
queue.Enqueue(sleepCommand{2s})
queue.Enqueue(cmdDevice.Hello())
queue.Enqueue(cmdStorage.GetFileInfo(path))
queue.Enqueue(cmdStorage.PlayVideo(path, true))
queue.Enqueue(cmdPreUpdate)
queue.Enqueue(cmdBright.SetBrightness(...))
queue.Enqueue(cmdPayload.SendOverlay(background))

// 5. Iniciar worker + sensors
```

### 3. Conversão de vídeo (FFmpeg)

**Quando**: Antes do upload, se o arquivo local não está no formato correto.

**Formato do device**: H.264 Main/High, level 3.0, 800×480, 24fps, yuv420p, sem áudio, MP4.

**Comando**:
```bash
ffmpeg -i input.mp4 -c:v libx264 -profile:v main -level 3.0 \
       -pix_fmt yuv420p -s 800x480 -r 24 -an \
       -movflags +faststart output.mp4
```

**Implementação**: Pode ser um passo manual (documentar) ou chamar `exec.Command("ffmpeg", ...)` no Go.

---

## Descobertas Críticas

1. **DTR=1, RTS=1** — OBRIGATÓRIO. Sem isso, `write()` bloqueia no Linux. Setado em `NewSerial()`.

2. **CMD_0x7D** (`7d ef 69 00 00 00 05 00 00 00 <brightness> 00`) — enviado antes do upload.

3. **Upload = write único** — `port.Write(data)` de uma vez. Sem chunking de 250 bytes.

4. **`file_rev_done` demora 5-15s** — polling com read timeout de 1s em loop até 60s.

5. **Overlay 0xca** (vídeo) vs **Bitmap 0xc8** (estático) — respostas diferentes.

6. **Sensor updates (0xcc)** — IGUAIS nos dois fluxos.

7. **`go.bug.st/serial`** — lib correta, suporta DTR/RTS e SetReadTimeout.

8. **Formato vídeo**: H.264, 800×480, 24fps, sem áudio, MP4.

9. **PCAP de referência**: `Debug/upload_full.pcapng` — 4 uploads completos.

10. **Scripts Python funcionam** — usar como referência (`scripts/turing_common.py`).

---

## Arquivos Relevantes

| Arquivo | Função |
|---------|--------|
| `cmd/turing-screen/main.go` | Entry point, orquestra fluxos |
| `internal/resource/serial/main.go` | Serial driver (go.bug.st/serial, DTR/RTS, WriteRaw, ReadPoll) |
| `internal/domain/command/storage.go` | CREATE_FILE, GET_FILE_INFO, PLAY_VIDEO, CMD_0x7D |
| `internal/domain/command/payload.go` | SendPayload (0xc8) e SendOverlay (0xca) |
| `internal/domain/command/pre_update_bitmap.go` | PRE_UPDATE_BITMAP (0x86) |
| `internal/domain/service/video/player.go` | PlayCommands(), UploadCommands(), DevicePath() |
| `internal/domain/service/sender/main.go` | Worker que processa a queue |
| `PROTOCOL.md` | Documentação completa do protocolo |
| `scripts/turing_common.py` | Implementação de referência Python (FUNCIONA) |
| `scripts/upload_and_play.py` | Upload + Play funcional em Python |
| `res/themes/NZXT_C/theme.yaml` | Tema de teste com `video_play` |

---

## Ordem de Execução

1. **Validar Fluxo 1** (estático) com `go.bug.st/serial` — confirmar que a troca de lib não quebrou nada
2. **Implementar Fluxo 2** — pré-fluxo síncrono + play via queue
3. **Testar com vídeo já no device** (pular upload) — confirmar play + overlay
4. **Implementar upload** — WriteRaw + ReadPoll
5. **Testar upload + play** completo
6. **Conversão FFmpeg** (opcional, pode ser manual)
