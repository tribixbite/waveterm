// Copyright 2024, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// ijson values are regular JSON values: string, number, boolean, null, object, array
// path is an array of strings and numbers

type PathType = (string | number)[];

var simplePathStrRe = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

function formatPath(path: PathType): string {
    if (path.length == 0) {
        return "$";
    }
    let pathStr = "$";
    for (let pathPart of path) {
        if (typeof pathPart === "string") {
            if (simplePathStrRe.test(pathPart)) {
                pathStr += "." + pathPart;
            } else {
                pathStr += "[" + JSON.stringify(pathPart) + "]";
            }
        } else {
            pathStr += "[" + pathPart + "]";
        }
    }
    return pathStr;
}

function getPath(obj: any, path: PathType): any {
    let cur = obj;
    for (let pathPart of path) {
        if (cur == null) {
            return null;
        }
        if (typeof pathPart === "string") {
            if (cur instanceof Object) {
                cur = cur[pathPart];
            } else {
                return null;
            }
        } else if (typeof pathPart === "number") {
            if (Array.isArray(cur)) {
                cur = cur[pathPart];
            } else {
                return null;
            }
        } else {
            throw new Error("Invalid path part: " + pathPart);
        }
    }
    return cur;
}

type SetPathOpts = {
    force?: boolean;
    remove?: boolean;
};

function setPath(obj: any, path: PathType, value: any, opts: SetPathOpts) {
    if (opts == null) {
        opts = {};
    }
    if (opts.remove && value != null) {
        throw new Error("Cannot set value and remove at the same time");
    }
    setPathInternal(obj, path, value, opts);
}

function isEmpty(obj: any): boolean {
    if (obj == null) {
        return true;
    }
    if (obj instanceof Object) {
        for (let key in obj) {
            return false;
        }
        return true;
    }
    return false;
}

function removeFromArr(arr: any[], idx: number): any[] {
    if (idx >= arr.length) {
        return arr;
    }
    if (idx == arr.length - 1) {
        arr.pop();
        return arr;
    }
    arr[idx] = null;
    return arr;
}

function setPathInternal(obj: any, path: PathType, value: any, opts: SetPathOpts): any {
    if (path.length == 0) {
        return value;
    }
    const pathPart = path[0];
    if (typeof pathPart === "string") {
        if (obj == null) {
            if (opts.remove) {
                return null;
            }
            obj = {};
        }
        if (!(obj instanceof Object)) {
            if (opts.force) {
                obj = {};
            } else {
                throw new Error("Cannot set path on non-object: " + obj);
            }
        }
        if (opts.remove && path.length == 1) {
            delete obj[pathPart];
            if (isEmpty(obj)) {
                return null;
            }
            return obj;
        }
        const newVal = setPath(obj[pathPart], path.slice(1), value, opts);
        if (opts.remove && newVal == null) {
            delete obj[pathPart];
            if (isEmpty(obj)) {
                return null;
            }
            return obj;
        }
        obj[pathPart] = newVal;
    } else if (typeof pathPart === "number") {
        if (pathPart < 0 || !Number.isInteger(pathPart)) {
            throw new Error("Invalid path part: " + pathPart);
        }
        if (obj == null) {
            if (opts.remove) {
                return null;
            }
            obj = [];
        }
        if (!Array.isArray(obj)) {
            if (opts.force) {
                obj = [];
            } else {
                throw new Error("Cannot set path on non-array: " + obj);
            }
        }
        if (opts.remove && path.length == 1) {
            return removeFromArr(obj, pathPart);
        }
        const newVal = setPath(obj[pathPart], path.slice(1), value, opts);
        if (opts.remove && newVal == null) {
            return removeFromArr(obj, pathPart);
        }
        obj[pathPart] = newVal;
    } else {
        throw new Error("Invalid path part: " + pathPart);
    }
}
