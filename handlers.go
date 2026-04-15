package main

/*
#include <stdlib.h>
#include <stdbool.h>
extern void* OpenPhysicalWindow(char* name, char* type, char* source, int w, int h, char* baseURI);
extern bool ClosePhysicalWindow(char* name);
extern void OpenNativeFilePicker(void* web_view_ptr);
extern void OpenNativeFolderPicker(void* web_view_ptr);
extern void ShowNativeAlert(void* web_view_ptr, char* title, char* message);
extern void ShowNativeConfirm(void* web_view_ptr, char* title, char* message);
extern bool RunJavaScriptByWindowName(char* name, char* js_code); 
*/
import "C"

import (
    "encoding/base64"
    "fmt"
    "os"
    "path/filepath"
    "time"
    "unsafe"
    "strings" // ADDED
    "encoding/json" // ADDED
    //"os/exec" // ADDED
    //"bytes" // Add this to your imports if it isn't there!
)
// The Whitelist lives here now - easy to find and edit
var funcWhitelist = map[string]FuncDefinition{
    "OpenNewWindow": {
        Handler: handleOpenNewWindow,
        ArgCount: 5,
        ArgTypes: []string{"string", "string", "string", "int", "int"},
    },
    "SayHello": {
        Handler: handleSayHello,
        ArgCount: 1,
        ArgTypes: []string{"string"},
    },
    "StringSend": {
        Handler: handleStringSend,
        ArgCount: 1,
        ArgTypes: []string{"string"},
    },
    "CloseWindow": {
        Handler: handleCloseNewWindow,
        ArgCount: 1,
        ArgTypes: []string{"string"},
    },
    "ReadFile": {
        Handler: handleReadFile, // Needs to be handleReadFile now
        ArgCount: 1,
        ArgTypes: []string{"string"},
    },
    "WriteFile": {
        Handler: handleWriteFile,
        ArgCount: 2,
        ArgTypes: []string{"string", "string"},
    },
    "DoesFileExist": {
        Handler: handleDoesFileExist,
        ArgCount: 1,
        ArgTypes: []string{"string"},
    },
    "HasTypeExtension": {
        Handler: handleHasTypeExtension,
        ArgCount: 2,
        ArgTypes: []string{"string", "string"},
    },
    "IsFolderLocationReal": {
        Handler: handleIsFolderLocationReal,
        ArgCount: 1,
        ArgTypes: []string{"string"},
    },
    "MakeDirectory": {
        Handler: handleMakeDirectory,
        ArgCount: 1,
        ArgTypes: []string{"string"},
    },
    "GetFolderContentsByPath": {
        Handler: handleGetFolderContentsByPath,
        ArgCount: 1,
        ArgTypes: []string{"string"},
    },
    "GetFolderFoldersByPath": {
        Handler: handleGetFolderFoldersByPath,
        ArgCount: 1,
        ArgTypes: []string{"string"},
    },
    "PickFile": {
        Handler: handlePickFile,
        ArgCount: 0,
        ArgTypes: []string{}, // No arguments needed from the frontend
    },
    "PickFolder": {
        Handler: handlePickFolder,
        ArgCount: 0,
        ArgTypes: []string{}, // No arguments needed from the frontend
    },
    "ShowMessage": {
        Handler: handleShowMessage,
        ArgCount: 2,
        ArgTypes: []string{"string", "string"},
    },
    "ConfirmMessage": {
        Handler: handleConfirmMessage,
        ArgCount: 2,
        ArgTypes: []string{"string", "string"},
    },
}

// Helper to resolve paths relative to the app root
// resolvePath finds the folder where the app binary sits and joins it with the relative path
func resolvePath(relPath string) (string, error) {
    // Get the absolute path of the running executable
    exePath, err := os.Executable()
    if err != nil {
        return "", err
    }

    // Get the directory containing that executable
    // Example: /home/user1/Programs/C/CGo/my_app -> /home/user1/Programs/C/CGo
    baseDir := filepath.Dir(exePath)

    // Join the binary's folder with the path provided by JS
    return filepath.Join(baseDir, relPath), nil
}


// --- Your Handlers ---
func handleSayHello(call JSCall) string {
    name := call.Args[0].(string)
    return fmt.Sprintf("Hello, %s!", name)
}

func handleReadFile(call JSCall) string {
    if len(call.Args) < 1 {
        return "false"
    }
    
    pathArg, ok := call.Args[0].(string)
    if !ok {
        return "false"
    }

    fullPath, err := resolvePath(pathArg)
    if err != nil {
        return "false"
    }

    data, err := os.ReadFile(fullPath)
    if err != nil {
        return "false"
    }
    
    return string(data)
}

func handleStringSend(call JSCall) string {
    if len(call.Args) < 1 {
        return "false"
    }

    // 1. Extract the JS code we want to run
    jsCode, ok := call.Args[0].(string)
    if !ok {
        return "false"
    }

    // 2. Identify the TARGET window (e.g., "Another Window")
    // We get this from the JSCall struct sent from the frontend
    targetWindow := call.WindowName 

    // 3. Convert to C types and fire across the bridge
    cTarget := C.CString(targetWindow)
    cJS := C.CString(jsCode)
    defer C.free(unsafe.Pointer(cTarget))
    defer C.free(unsafe.Pointer(cJS))

    // This calls the actual C routing logic in BridgeDev.c
    success := C.RunJavaScriptByWindowName(cTarget, cJS)

    return fmt.Sprintf("%t", bool(success))
}

func handleWriteFile(call JSCall) string {
    if len(call.Args) < 2 {
        return "false"
    }

    pathArg, okPath := call.Args[0].(string)
    content, okContent := call.Args[1].(string)

    if !okPath || !okContent {
        return "false"
    }

    fullPath, err := resolvePath(pathArg)
    if err != nil {
        return "false"
    }

    // Write file with standard permissions
    err = os.WriteFile(fullPath, []byte(content), 0644)
    if err != nil {
        return "false"
    }
    
    return "true"
}

// Then your handler becomes very slim:
func handleDoesFileExist(call JSCall) string {
    if len(call.Args) < 1 {
        return "false"
    }

    pathArg, ok := call.Args[0].(string)
    if !ok {
        return "false"
    }

    // Use the same helper to anchor the path to the executable's folder
    fullPath, err := resolvePath(pathArg)
    if err != nil {
        return "false"
    }

    // os.Stat returns info about the file; if err is nil, the file exists
    _, err = os.Stat(fullPath)
    if err == nil {
        return "true"
    }

    // If there's an error (like file not found), return "false"
    return "false"
}

func handleHasTypeExtension(call JSCall) string {
    if len(call.Args) < 2 {
        return "false"
    }

    pathArg, ok1 := call.Args[0].(string)
    extArg, ok2 := call.Args[1].(string)
    if !ok1 || !ok2 {
        return "false"
    }

    // Programmatically resolve the path relative to the executable
    fullPath, err := resolvePath(pathArg)
    if err != nil {
        return "false"
    }

    // Get the extension from the resolved path (e.g., ".txt")
    pathExt := filepath.Ext(fullPath)

    // Ensure our target extension has a dot for comparison
    targetExt := extArg
    if !strings.HasPrefix(extArg, ".") {
        targetExt = "." + extArg
    }

    // Compare case-insensitively
    if strings.EqualFold(pathExt, targetExt) {
        return "true"
    }
    return "false"
}

func handleIsFolderLocationReal(call JSCall) string {
    if len(call.Args) < 1 {
        return "false"
    }

    folderLocation, ok := call.Args[0].(string)
    if !ok {
        return "false"
    }

    // Programmatically resolve the path relative to the executable location
    fullPath, err := resolvePath(folderLocation)
    if err != nil {
        return "false"
    }

    // Get file/folder information
    info, err := os.Stat(fullPath)
    if err != nil {
        // Path doesn't exist or is inaccessible
        return "false"
    }

    // Check if the resolved path is specifically a directory
    if info.IsDir() {
        return "true"
    }

    return "false"
}

func handleMakeDirectory(call JSCall) string {
    if len(call.Args) < 1 {
        return "false"
    }

    directoryPath, ok := call.Args[0].(string)
    if !ok {
        return "false"
    }

    // Programmatically resolve the path relative to the executable
    fullPath, err := resolvePath(directoryPath)
    if err != nil {
        return "false"
    }

    // 0755: Owner can read/write/execute, others can read/execute
    // MkdirAll is idempotent; if the directory already exists, it returns nil
    err = os.MkdirAll(fullPath, 0755)
    if err != nil {
        return "false"
    }

    return "true"
}

func handleGetFolderContentsByPath(call JSCall) string {
    if len(call.Args) < 1 {
        return "false"
    }

    folderPath, ok := call.Args[0].(string)
    if !ok {
        return "false"
    }

    // Programmatically resolve the path relative to the executable
    fullPath, err := resolvePath(folderPath)
    if err != nil {
        return "false"
    }

    // Read the directory entries
    entries, err := os.ReadDir(fullPath)
    if err != nil {
        // Returns "false" if the directory doesn't exist or is inaccessible
        return "false"
    }

    var fileNames []string
    for _, entry := range entries {
        // Only append if it's a file, not a directory
        if !entry.IsDir() {
            fileNames = append(fileNames, entry.Name())
        }
    }

    // Serialize the slice of strings into a JSON array
    jsonData, err := json.Marshal(fileNames)
    if err != nil {
        return "false"
    }

    return string(jsonData)
}

func handleGetFolderFoldersByPath(call JSCall) string {
    if len(call.Args) < 1 {
        return "false"
    }

    folderPath, ok := call.Args[0].(string)
    if !ok {
        return "false"
    }

    // Programmatically resolve the path relative to the executable
    fullPath, err := resolvePath(folderPath)
    if err != nil {
        return "false"
    }

    // Read the directory contents
    entries, err := os.ReadDir(fullPath)
    if err != nil {
        return "false"
    }

    var folderNames []string
    for _, entry := range entries {
        // Change logic from previous function: only append if it IS a directory
        if entry.IsDir() {
            folderNames = append(folderNames, entry.Name())
        }
    }

    // Serialize the slice of folder names to a JSON string
    jsonData, err := json.Marshal(folderNames)
    if err != nil {
        return "false"
    }

    return string(jsonData)
}

func handlePickFile(call JSCall) string {
    var webViewPtr unsafe.Pointer
    for _, win := range GoOpenedWindows {
        if win.WindowName == call.WindowName {
            webViewPtr = win.webViewPtr
            break
        }
    }
    if webViewPtr != nil {
        // Just call it! No cPath := C.OpenNativeFilePicker()
        C.OpenNativeFilePicker(webViewPtr)
    }
    return "triggered"
}

func handlePickFolder(call JSCall) string {
    var webViewPtr unsafe.Pointer
    for _, win := range GoOpenedWindows {
        if win.WindowName == call.WindowName {
            webViewPtr = win.webViewPtr
            break
        }
    }
    if webViewPtr != nil {
        C.OpenNativeFolderPicker(webViewPtr)
    }
    return "triggered" 
}

func handleShowMessage(call JSCall) string {
    if len(call.Args) < 2 {
        return "false"
    }
    title, ok1 := call.Args[0].(string)
    message, ok2 := call.Args[1].(string)
    if !ok1 || !ok2 {
        return "false"
    }
    var webViewPtr unsafe.Pointer
    for _, win := range GoOpenedWindows {
        if win.WindowName == call.WindowName {
            webViewPtr = win.webViewPtr
            break
        }
    }
    if webViewPtr != nil {
        cTitle := C.CString(title)
        cMessage := C.CString(message)
        defer C.free(unsafe.Pointer(cTitle))
        defer C.free(unsafe.Pointer(cMessage))

        C.ShowNativeAlert(webViewPtr, cTitle, cMessage)
    }
    return "true"
}

func handleConfirmMessage(call JSCall) string {
    if len(call.Args) < 2 {
        return "false"
    }

    title, ok1 := call.Args[0].(string)
    message, ok2 := call.Args[1].(string)

    if !ok1 || !ok2 {
        return "false"
    }

    var webViewPtr unsafe.Pointer
    for _, win := range GoOpenedWindows {
        if win.WindowName == call.WindowName {
            webViewPtr = win.webViewPtr
            break
        }
    }

    if webViewPtr != nil {
        cTitle := C.CString(title)
        cMessage := C.CString(message)
        defer C.free(unsafe.Pointer(cTitle))
        defer C.free(unsafe.Pointer(cMessage))

        C.ShowNativeConfirm(webViewPtr, cTitle, cMessage)
    }

    return "triggered"
}

//-------------------------------------------------

func handleOpenNewWindow(call JSCall) string {
    if len(call.Args) < 5 { return "false" }

    // Type assertions
    name, _ := call.Args[0].(string)
    addrType, _ := call.Args[1].(string)
    source, _ := call.Args[2].(string)
    w := int(call.Args[3].(float64))
    h := int(call.Args[4].(float64))

    finalSource := source
    if addrType == "HTMLString" {
        decoded, err := base64.StdEncoding.DecodeString(source)
        if err == nil {
            finalSource = string(decoded)
        }
    }

    // 1. Get Executable Directory
    exePath, err := os.Executable()
    if err != nil { return "false" }
    exeDir := filepath.Dir(exePath)

    // 2. Format the BaseURI properly
    // We use filepath.ToSlash to ensure / even on Windows
    // We use file:/// (triple slash) for a valid local URI
    formattedDir := filepath.ToSlash(exeDir)
    if !strings.HasPrefix(formattedDir, "/") {
        formattedDir = "/" + formattedDir
    }
    baseURI := "file://" + formattedDir + "/"

    // 3. If it's an HTMLAddress (a file), ensure the source is an absolute path
    // Many webviews fail if you pass a relative path as the 'source'
    if addrType == "HTMLAddress" {
        if !filepath.IsAbs(finalSource) {
            finalSource = filepath.Join(exeDir, finalSource)
            finalSource = "file://" + filepath.ToSlash(finalSource)
        }
    }

    // C Conversions
    cName := C.CString(name)
    cType := C.CString(addrType)
    cSource := C.CString(finalSource)
    cBaseURI := C.CString(baseURI)
    defer C.free(unsafe.Pointer(cName))
    defer C.free(unsafe.Pointer(cType))
    defer C.free(unsafe.Pointer(cSource))
    defer C.free(unsafe.Pointer(cBaseURI))

    ptr := C.OpenPhysicalWindow(cName, cType, cSource, C.int(w), C.int(h), cBaseURI)
    if ptr == nil { return "false" }

    // Track the window
    GoOpenedWindows = append(GoOpenedWindows, GoOpenedWindow{
        WindowName: name,
        TimeStamp:  time.Now().Format("2006-01-02 15:04:05"),
        Width:      w,
        Height:     h,
        webViewPtr: ptr,
    })

    return "true"
}

func handleCloseNewWindow(call JSCall) string {
    if len(call.Args) < 1 { return "false" }
    targetName := call.Args[0].(string)
    indexNum := -1
    for i, win := range GoOpenedWindows {
        if win.WindowName == targetName {
            indexNum = i
            break
        }
    }
    if indexNum == -1 { return "false" }
    cName := C.CString(targetName)
    defer C.free(unsafe.Pointer(cName))
    if bool(C.ClosePhysicalWindow(cName)) {
        // Remove from slice (The Splice)
        GoOpenedWindows = append(GoOpenedWindows[:indexNum], GoOpenedWindows[indexNum+1:]...)
        return "true"
    }
    return "false"
}