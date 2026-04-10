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
*/
import "C" // <--- This MUST be here, right after the comment

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "strings"
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

// --- WHITELISTED HANDLERS ---

//export GoAppActivate
func GoAppActivate() {
    // 1. Get the working directory for asset pathing
    pwd, _ := os.Getwd()
    baseURI := "file://" + filepath.ToSlash(pwd) + "/"

    // 2. Define the start page
    windowName := C.CString("Main Window")
    addrType := C.CString("HTMLAddress")
    source := C.CString("file://" + filepath.Join(pwd, "index.html"))
    fileloc := C.CString(baseURI)

    defer C.free(unsafe.Pointer(windowName))
    defer C.free(unsafe.Pointer(addrType))
    defer C.free(unsafe.Pointer(source))
    defer C.free(unsafe.Pointer(fileloc))

    // 3. Kick off the first window AND capture the pointer
    ptr := C.OpenPhysicalWindow(windowName, addrType, source, 1024, 768, fileloc)

    // 4. Save the main window to the Go tracker!
    if ptr != nil {
        GoOpenedWindows = append(GoOpenedWindows, GoOpenedWindow{
            WindowName: "Main Window",
            TimeStamp:  time.Now().Format("2006-01-02 15:04:05"),
            Width:      1024,
            Height:     768,
            webViewPtr: ptr,
        })
    }
}

func validateType(val any, expected string) bool {
    switch expected {
        case "string":
        _, ok := val.(string)
        return ok
        case "int":

        // JSON numbers are float64. Check if it's a whole number.
        f, ok := val.(float64)
            return ok && f == float64(int(f))
        case "float64":
            _, ok := val.(float64)
            return ok
        case "array":
            _, ok := val.([]any)
            return ok
        case "object":
            _, ok := val.(map[string]any)
            return ok
        default:
            return false
    }
}

// --- TRAFFIC COP ---
type AuthorizedFunc func(JSCall) string
type FuncDefinition struct {
    Handler AuthorizedFunc
    ArgCount int
    ArgTypes []string
}

//export GoTrafficCop
func GoTrafficCop(rawJSON *C.char) *C.char {
    input := C.GoString(rawJSON)
    var call JSCall
    if err := json.Unmarshal([]byte(input), &call); err != nil {
        return createErrorResponse(0, "Packet Mangle Error")
    }

    // 1. Check if function exists
    def, exists := funcWhitelist[call.FuncName]
    if !exists {
        return createErrorResponse(call.ID, "Unauthorized Call: "+call.FuncName)
    }

    // 2. SAFETY FIRST: Check lengths before accessing array indices
    // This prevents a panic if call.Args is shorter than expected.
    if len(call.Args) != def.ArgCount || len(call.Types) != def.ArgCount {
        return createErrorResponse(call.ID, fmt.Sprintf("Argument Count Mismatch: Expected %d", def.ArgCount))
    }

    // 3. Single Loop for Type Validation
    for i, expectedType := range def.ArgTypes {
        // Check the string label sent from JS ("string", "int", etc)
        if call.Types[i] != expectedType {
            return createErrorResponse(call.ID, fmt.Sprintf("Type Definition Mismatch: Arg[%d] should be %s", i, expectedType))
        }

        // Check the ACTUAL underlying Go data (the helper we wrote)
        if !validateType(call.Args[i], expectedType) {
            return createErrorResponse(call.ID, fmt.Sprintf("Data Integrity Error: Arg[%d] is not a valid %s", i, expectedType))
        }
    }
    
    // 4. Address Security Check (The "Adr" prefix)
    if !strings.HasPrefix(call.ArbAddress, "Adr") {
        return createErrorResponse(call.ID, "Security Error: Invalid Address Prefix")
    }

    // 5. THE ROUTER (Traffic Cop doing actual traffic directing)
    // If the targeted WindowName is NOT empty and does NOT match the sending window,
    // we bypass the handlers and execute directly on that target window!
    if call.WindowName != "" && call.WindowName != "Main Window" { // Replace "Main Window" with your origin identifier if needed
        // We expect the script to execute to be the first argument
        if len(call.Args) > 0 {
            jsCommand, ok := call.Args[0].(string)
            if ok {
                cTarget := C.CString(call.WindowName)
                cJS := C.CString(jsCommand)
                defer C.free(unsafe.Pointer(cTarget))
                defer C.free(unsafe.Pointer(cJS))
                success := C.RunJavaScriptByWindowName(cTarget, cJS)
                resp := GoResponse{ID: call.ID, Result: fmt.Sprintf("%t", bool(success))}
                jsonBytes, _ := json.Marshal(resp)
                return C.CString(base64.StdEncoding.EncodeToString(jsonBytes))
            }
        }
    }

    // 6. Normal Execution (If it was meant for the local window or background task)
    result := def.Handler(call)
    resp := GoResponse{ID: call.ID, Result: result}
    jsonBytes, _ := json.Marshal(resp)
    return C.CString(base64.StdEncoding.EncodeToString(jsonBytes))
}

// Helper to keep the main function clean
func createErrorResponse(id int, message string) *C.char {
    resp := GoResponse{ID: id, Result: "Error: " + message}
    jsonBytes, _ := json.Marshal(resp)
    return C.CString(base64.StdEncoding.EncodeToString(jsonBytes))
}

func main() {}