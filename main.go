package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type appState struct {
	lines        []string
	filename     string
	startLine    int // 视图起始行
	startCol     int // 视图起始列
	currentLine  int // 光标所在行
	currentCol   int // 光标所在列
	screenWidth  int
	screenHeight int
	lastGPress   bool
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <filename>", os.Args[0])
	}

	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(string(content), "\n")
	state := &appState{
		lines:       lines,
		filename:    os.Args[1],
		currentLine: 0,
		currentCol:  0,
		startLine:   0,
		startCol:    0,
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		log.Fatal(err)
	}
	defer screen.Fini()

mainLoop:
	for {
		w, h := screen.Size()
		state.screenWidth = w
		state.screenHeight = h
		contentHeight := h - 3 // 内容区域高度

		// 自动调整垂直视图确保光标可见（仅在键盘操作时）
		if state.currentLine < state.startLine {
			state.startLine = state.currentLine
		}
		if state.currentLine >= state.startLine+contentHeight {
			state.startLine = state.currentLine - contentHeight + 1
		}

		// 水平边界检查
		currentLineLength := len(state.lines[state.currentLine])
		if state.currentCol > currentLineLength {
			state.currentCol = currentLineLength
		}

		drawUI(screen, state)

		switch ev := screen.PollEvent().(type) {
		case *tcell.EventResize:
			screen.Sync()
		case *tcell.EventMouse:
			x, y := ev.Position()
			buttons := ev.Buttons()

			// 处理鼠标滚轮
			switch {
			case buttons&tcell.WheelUp != 0:
				state.startLine -= 3
				if state.startLine < 0 {
					state.startLine = 0
				}
			case buttons&tcell.WheelDown != 0:
				maxStart := len(state.lines) - contentHeight
				if maxStart < 0 {
					maxStart = 0
				}
				state.startLine += 3
				if state.startLine > maxStart {
					state.startLine = maxStart
				}
			case buttons&tcell.Button1 != 0: // 左键点击
				// 转换点击坐标到文档位置
				if y >= 1 && y < h-2 { // 在内容区域
					clickLine := state.startLine + (y - 1)
					if clickLine < len(state.lines) {
						state.currentLine = clickLine
						// 计算列位置
						clickCol := state.startCol + (x - 1)
						if clickCol > len(state.lines[clickLine]) {
							clickCol = len(state.lines[clickLine])
						}
						if clickCol >= 0 {
							state.currentCol = clickCol
						}
					}
				}
			}
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlC:
				break mainLoop
			case tcell.KeyDown:
				moveCursor(state, 1, 0)
			case tcell.KeyUp:
				moveCursor(state, -1, 0)
			case tcell.KeyRight:
				moveCursor(state, 0, 1)
			case tcell.KeyLeft:
				moveCursor(state, 0, -1)
			case tcell.KeyCtrlB: // 上翻整页
				state.startLine -= contentHeight
				if state.startLine < 0 {
					state.startLine = 0
				}
			case tcell.KeyCtrlF: // 下翻整页
				state.startLine += contentHeight
				maxStart := len(state.lines) - contentHeight
				if maxStart < 0 {
					maxStart = 0
				}
				if state.startLine > maxStart {
					state.startLine = maxStart
				}
			case tcell.KeyCtrlD: // 下翻半页
				state.startLine += contentHeight / 2
				maxStart := len(state.lines) - contentHeight
				if maxStart < 0 {
					maxStart = 0
				}
				if state.startLine > maxStart {
					state.startLine = maxStart
				}
			case tcell.KeyCtrlU: // 上翻半页
				state.startLine -= contentHeight / 2
				if state.startLine < 0 {
					state.startLine = 0
				}
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'j':
					moveCursor(state, 1, 0)
				case 'k':
					moveCursor(state, -1, 0)
				case 'h':
					moveCursor(state, 0, -1)
				case 'l':
					moveCursor(state, 0, 1)
				case 'q':
					break mainLoop
				case 'g':
					if state.lastGPress {
						state.currentLine = 0
						state.lastGPress = false
					} else {
						state.lastGPress = true
					}
				case 'G':
					state.currentLine = len(state.lines) - 1
					state.lastGPress = false
				}
			}
		}
	}
}

func moveCursor(state *appState, dy, dx int) {
	// 垂直移动
	if dy != 0 {
		newLine := state.currentLine + dy
		if newLine >= 0 && newLine < len(state.lines) {
			state.currentLine = newLine
		}
	}

	// 水平移动
	if dx != 0 {
		currentLineLength := len(state.lines[state.currentLine])
		newCol := state.currentCol + dx
		if newCol >= 0 && newCol <= currentLineLength {
			state.currentCol = newCol
		}
	}
}

func drawUI(screen tcell.Screen, state *appState) {
	screen.Clear()
	w, h := state.screenWidth, state.screenHeight

	// 绘制边框
	drawBorder(screen, w, h)

	// 绘制内容
	contentHeight := h - 3
	for i := 0; i < contentHeight; i++ {
		lineNum := state.startLine + i
		if lineNum >= len(state.lines) {
			break
		}
		line := state.lines[lineNum]
		start := state.startCol
		end := start + (w - 2)
		if end > len(line) {
			end = len(line)
		}
		if start < len(line) {
			drawString(screen, 1, 1+i, line[start:end])
		}
	}

	// 状态栏
	status := fmt.Sprintf(" %s | Line: %d/%d | Col: %d ",
		state.filename,
		state.currentLine+1,
		len(state.lines),
		state.currentCol+1,
	)
	if len(status) > w-2 {
		status = status[:w-2]
	}
	drawString(screen, 1, h-2, status)

	// 光标位置
	relY := state.currentLine - state.startLine
	relX := state.currentCol - state.startCol
	if relY >= 0 && relY < contentHeight && relX >= 0 && relX < w-2 {
		screen.ShowCursor(relX+1, relY+1)
	} else {
		screen.HideCursor()
	}

	screen.Show()
}

func drawBorder(screen tcell.Screen, w, h int) {
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite)

	// 绘制四角
	screen.SetContent(0, 0, '┌', nil, style)
	screen.SetContent(w-1, 0, '┐', nil, style)
	screen.SetContent(0, h-1, '└', nil, style)
	screen.SetContent(w-1, h-1, '┘', nil, style)

	// 绘制边框
	for x := 1; x < w-1; x++ {
		screen.SetContent(x, 0, '─', nil, style)
		screen.SetContent(x, h-2, '─', nil, style)
		screen.SetContent(x, h-1, ' ', nil, style)
	}

	for y := 1; y < h-2; y++ {
		screen.SetContent(0, y, '│', nil, style)
		screen.SetContent(w-1, y, '│', nil, style)
	}
}

func drawString(screen tcell.Screen, x, y int, str string) {
	for i, c := range str {
		screen.SetContent(x+i, y, c, nil, tcell.StyleDefault)
	}
}

func moveVertical(state *appState, delta int) {
	newLine := state.currentLine + delta
	if newLine >= 0 && newLine < len(state.lines) {
		state.currentLine = newLine
		// 移动到新行时调整列位置
		if state.currentCol >= len(state.lines[state.currentLine]) {
			state.currentCol = len(state.lines[state.currentLine]) - 1
			if state.currentCol < 0 {
				state.currentCol = 0
			}
		}
	}
}

func moveHorizontal(state *appState, delta int) {
	newCol := state.currentCol + delta
	if newCol >= 0 && newCol < len(state.lines[state.currentLine]) {
		state.currentCol = newCol
	}
}

func pageDown(state *appState, lines int) {
	newLine := state.currentLine + lines
	if newLine >= len(state.lines) {
		newLine = len(state.lines) - 1
	}
	state.currentLine = newLine
}

func pageUp(state *appState, lines int) {
	newLine := state.currentLine - lines
	if newLine < 0 {
		newLine = 0
	}
	state.currentLine = newLine
}
