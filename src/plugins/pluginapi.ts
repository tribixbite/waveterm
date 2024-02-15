// Copyright 2024, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

type CmdContextType = {
    screenid: string;
    lineid: string;
    linenum: number;
    ts: number;
    cmdstr: string;
    festate: Record<string, string>;
    remoteptr: RemotePtrType;
    cmdstatus: string;
    exitcode: number;
    durationms: number;
};

type DataUpdateType = {
    pos: number;
    data: Uint8Array;
    eof: boolean;
};

type PluginApi = {
    loadPtyData(): Promise<ExtFile>;
    onDataUpdate(datatype: string, cb: (datatype: string, update: DataUpdateType) => void): void;
    onCmdDone(cb: (cmdContext: CmdContextType) => void): void;
    releaseFocus(): void;
    setLineState(lineState: LineStateType): void;
    writeRemoteFile(filePath: string, data: Uint8Array): Promise<void>;
    readRemoteFile(filePath: string): ExtFile;
    streamRemoteFile(filePath: string, cb: (data: DataUpdateType) => void): void;
};

type PluginProps = {
    cmdContext: CmdContextType;
    pluginDecl: RendererPluginType;
    rendererOpts: RendererOpts;
    lineState: LineStateType;
    focusState: "none" | "selected" | "focused";
    api: PluginApi;
};
