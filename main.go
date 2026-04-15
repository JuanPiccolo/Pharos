package main 

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.1
#include <stdlib.h>
#include <stdbool.h>
extern void* OpenPhysicalWindow(char* name, char* type, char* source, int w, int h, char* fileloc);
extern bool ClosePhysicalWindow(char* name);
extern void RunJavaScriptInWindow(void* web_view_ptr, char* js_code);
extern bool RunJavaScriptByWindowName(char* name, char* js_code);
extern void OpenNativeFilePicker(void* web_view_ptr);
extern void GoWindowClosedNotify(char* name);
*/
import "C" // <--- This MUST be here, right after the comment

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    //"strings"
    "os"
    "path/filepath"
    "time"
    "unsafe"
)

// --- BRIDGE TYPES ---

type JSCall struct {
    ID int `json:"id"`
    FuncName string `json:"funcName"`
    ArbAddress string `json:"arbAddress"`
    WindowName string `json:"windowName"`
    Args []any `json:"args"`
    Types []string `json:"types"`
}

type GoResponse struct {
    ID int `json:"id"`
    Result string `json:"result"`
}

type GoOpenedWindow struct {
    WindowName string `json:"windowName"`
    TimeStamp string `json:"timeStamp"`
    Width int `json:"width"`
    Height int `json:"height"`
    webViewPtr unsafe.Pointer
}

var GoOpenedWindows []GoOpenedWindow

const MainWindowLabel = "Main Window"

//export GoAppActivate
func GoAppActivate() {
    // 1. Get the working directory for asset pathing
    pwd, _ := os.Getwd()
    baseURI := "file://" + filepath.ToSlash(pwd) + "/"

    // 2. Define the start page
    windowName := C.CString(MainWindowLabel)
    addrType := C.CString("HTMLAddress")
    source := C.CString("file://" + filepath.Join(pwd, "index.html"))
    fileloc := C.CString(baseURI)

    defer C.free(unsafe.Pointer(windowName))
    defer C.free(unsafe.Pointer(addrType))
    defer C.free(unsafe.Pointer(source))
    defer C.free(unsafe.Pointer(fileloc))

    // 3. Kick off the first window AND capture the pointer
    ptr := C.OpenPhysicalWindow(windowName, addrType, source, 675, 775, fileloc)

    // 4. Save the main window to the Go tracker!
    if ptr != nil {
        GoOpenedWindows = append(GoOpenedWindows, GoOpenedWindow{
            WindowName: MainWindowLabel,
            TimeStamp:  time.Now().Format("2006-01-02 15:04:05"),
            Width:      675,
            Height:     775,
            webViewPtr: ptr,
        })
    }
}


func createResponse(id int, result string) *C.char {
	resp := GoResponse{ID: id, Result: result}
	jsonBytes, _ := json.Marshal(resp)
	// Base64 protects the JSON string during the C-Bridge crossing
	return C.CString(base64.StdEncoding.EncodeToString(jsonBytes))
}

func createErrorResponse(id int, message string) *C.char {
	return createResponse(id, "Error: "+message)
}

//export GoTrafficCop
func GoTrafficCop(rawJSON *C.char) *C.char {
	input := C.GoString(rawJSON)
	
	var call JSCall
	if err := json.Unmarshal([]byte(input), &call); err != nil {
		return createErrorResponse(0, "Packet Mangle Error")
	}

	// 1. DIRECTION A: REMOTE ROUTING
	// Logic: If targeting another window, bypass local Go handlers.
	if call.FuncName == "ExecuteRemoteJS" {
		if len(call.Args) > 0 {
			jsCommand, ok := call.Args[0].(string)
			if ok {
				cTarget := C.CString(call.WindowName)
				cJS := C.CString(jsCommand)
				
				success := C.RunJavaScriptByWindowName(cTarget, cJS)
				
				// Standard C memory management
				C.free(unsafe.Pointer(cTarget)) 
				C.free(unsafe.Pointer(cJS))

				return createResponse(call.ID, fmt.Sprintf("%t", bool(success)))
			}
		}
	}

	// 2. DIRECTION B: LOCAL HANDLER EXECUTION
	def, exists := funcWhitelist[call.FuncName]
	if !exists {
		return createErrorResponse(call.ID, "Unauthorized Call: "+call.FuncName)
	}

	if len(call.Args) != def.ArgCount {
		return createErrorResponse(call.ID, "Argument Mismatch")
	}

	// Execute the designated Go handler (e.g., ReadFile)
	result := def.Handler(call)
	return createResponse(call.ID, result)
}

//export GoWindowClosedNotify
func GoWindowClosedNotify(rawName *C.char) {
    name := C.GoString(rawName)
    
    // Find and remove from the tracker slice
    for i, win := range GoOpenedWindows {
        if win.WindowName == name {
            // Standard slice deletion (The Splice)
            GoOpenedWindows = append(GoOpenedWindows[:i], GoOpenedWindows[i+1:]...)
            fmt.Printf("[Go] State Cleaned: Window '%s' removed from tracker.\n", name)
            break
        }
    }
}

// --- TRAFFIC COP ---
type AuthorizedFunc func(JSCall) string
type FuncDefinition struct {
    Handler AuthorizedFunc
    ArgCount int
    ArgTypes []string
}

func main() {}