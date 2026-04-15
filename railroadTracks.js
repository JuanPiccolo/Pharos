//#region Do not cross the tracks!
/*
===============================================================================
|||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
===============================================================================*/
let id_Count = 0;
// A place to store "Pending" calls
const pendingCalls = new Map();

async function CallCGo(FunctionName, Address, WindowName, args, types) {
    let id = id_Count++;
    // 1. Create a promise that we will resolve later
    return new Promise((resolve, reject) => {
        // Store the resolve function so we can call it by ID later
        pendingCalls.set(id, { resolve, reject });
        const payload = {
            id: id,
            funcName: FunctionName,
            arbAddress: Address,
            windowName: WindowName,
            args: args,
            types: types
        };
        window.webkit.messageHandlers.c_bridge.postMessage(JSON.stringify(payload));
    });
}

// 2. This is the global function C will call when Go is done
/*
function ReceiveFromGo(jsonResponse) {
const data = JSON.parse(jsonResponse);
if (pendingCalls.has(data.id)) {
// Retrieve the stored resolve function and trigger it!
const { resolve } = pendingCalls.get(data.id);
resolve(data.result);
pendingCalls.delete(data.id); // Clean up
}
}
*/

function ReceiveFromGo(base64Data) {
    // 1. Decode from Base64 to a JSON string
    const jsonString = atob(base64Data);

    // 2. Parse the JSON string into an object
    const data = JSON.parse(jsonString);

    // 3. Route it to your pending calls
    if (pendingCalls.has(data.id)) {
        const { resolve } = pendingCalls.get(data.id);
        resolve(data.result);
        pendingCalls.delete(data.id);
    }
}

/*
===============================================================================
|||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
===============================================================================*/
//#endregion Do not cross the tracks!