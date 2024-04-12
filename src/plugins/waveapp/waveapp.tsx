// Copyright 2024, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import * as React from "react";
import * as mobx from "mobx";
import * as mobxReact from "mobx-react";
import { debounce } from "throttle-debounce";
import { boundMethod } from "autobind-decorator";
import { JsonLinesDataBuffer } from "../core/ptydata";
import { Markdown } from "@/elements";
import * as ijson from "@/util/ijson";

import "./waveapp.less";

type WaveAppProps = {
    lineId: string;
    isSelected: boolean;
    isFocused: boolean;
    savedHeight: number;
    initialData: any;
    onPacket: (packetFn: (packet: any) => void) => void;
};

type WaveAppNode = {
    tag: string;
    props?: Record<string, any>;
    children?: (WaveAppNode | string)[];
};

const TagMap: Record<string, React.ComponentType<{ node: WaveAppNode }>> = {};

function convertNodeToTag(node: WaveAppNode | string, idx?: number): JSX.Element | string {
    if (node == null) {
        return null;
    }
    if (idx == null) {
        idx = 0;
    }
    if (typeof node === "string") {
        return node;
    }
    let key = node.props?.key ?? "child-" + idx;
    let TagComp = TagMap[node.tag];
    if (!TagComp) {
        return (
            <div key={key} s>
                Unknown tag:{node.tag}
            </div>
        );
    }
    return <TagComp key={key} node={node} />;
}

@mobxReact.observer
class WaveAppHtmlTag extends React.Component<{ node: WaveAppNode }, {}> {
    render() {
        let { tag, props, children } = this.props.node;
        let divProps = {};
        if (props != null) {
            for (let [key, val] of Object.entries(props)) {
                if (key.startsWith("on")) {
                    divProps[key] = (e: any) => {
                        console.log("handler", key, val);
                    };
                } else {
                    divProps[key] = mobx.toJS(val);
                }
            }
        }
        let childrenComps = [];
        if (children != null) {
            for (let idx = 0; idx < children.length; idx++) {
                let comp = convertNodeToTag(children[idx], idx);
                if (comp != null) {
                    childrenComps.push(comp);
                }
            }
        }
        return React.createElement(tag, divProps, childrenComps);
    }
}

TagMap["div"] = WaveAppHtmlTag;
TagMap["b"] = WaveAppHtmlTag;
TagMap["i"] = WaveAppHtmlTag;
TagMap["p"] = WaveAppHtmlTag;
TagMap["span"] = WaveAppHtmlTag;
TagMap["a"] = WaveAppHtmlTag;
TagMap["h1"] = WaveAppHtmlTag;
TagMap["h2"] = WaveAppHtmlTag;
TagMap["h3"] = WaveAppHtmlTag;
TagMap["h4"] = WaveAppHtmlTag;
TagMap["h5"] = WaveAppHtmlTag;
TagMap["h6"] = WaveAppHtmlTag;
TagMap["ul"] = WaveAppHtmlTag;
TagMap["ol"] = WaveAppHtmlTag;
TagMap["li"] = WaveAppHtmlTag;

class WaveAppRendererModel {
    context: RendererContext;
    opts: RendererOpts;
    isDone: OV<boolean>;
    api: RendererModelContainerApi;
    savedHeight: number;
    loading: OV<boolean>;
    ptyDataSource: (termContext: TermContextUnion) => Promise<PtyDataType>;
    packetData: JsonLinesDataBuffer;
    rawCmd: WebCmd;
    version: OV<number>;
    loadError: OV<string> = mobx.observable.box(null, { name: "renderer-loadError" });
    data: OV<any> = mobx.observable.box(null, { name: "renderer-data" });

    constructor() {
        this.packetData = new JsonLinesDataBuffer(this.packetCallback.bind(this));
        this.version = mobx.observable.box(0);
    }

    initialize(params: RendererModelInitializeParams): void {
        this.loading = mobx.observable.box(true, { name: "renderer-loading" });
        this.isDone = mobx.observable.box(params.isDone, { name: "renderer-isDone" });
        this.context = params.context;
        this.opts = params.opts;
        this.api = params.api;
        this.savedHeight = params.savedHeight;
        this.ptyDataSource = params.ptyDataSource;
        this.rawCmd = params.rawCmd;
        setTimeout(() => this.reload(0), 10);
    }

    packetCallback(jsonVal: any) {
        console.log("packet-callback", jsonVal);
        try {
            let data = this.data.get();
            let newData = ijson.applyCommand(data, jsonVal);
            console.log("got newdata", newData);
            if (newData != data) {
                mobx.action(() => {
                    this.data.set(newData);
                })();
            }
        } catch (e) {
            console.log("error adding data", e);
        }
        return;
    }

    dispose(): void {
        return;
    }

    reload(delayMs: number): void {
        mobx.action(() => {
            this.loading.set(true);
            this.loadError.set(null);
        })();
        let rtnp = this.ptyDataSource(this.context);
        if (rtnp == null) {
            console.log("no promise returned from ptyDataSource (waveapp renderer)", this.context);
            return;
        }
        rtnp.then((ptydata) => {
            setTimeout(() => {
                this.packetData.reset();
                this.receiveData(ptydata.pos, ptydata.data, "reload");
                mobx.action(() => {
                    this.loading.set(false);
                })();
            }, delayMs);
        }).catch((e) => {
            console.log("error loading data", e);
            mobx.action(() => {
                this.loadError.set("error loading data: " + e);
            })();
        });
    }

    giveFocus(): void {
        return;
    }

    updateOpts(opts: RendererOptsUpdate): void {
        Object.assign(this.opts, opts);
    }

    setIsDone(): void {
        if (this.isDone.get()) {
            return;
        }
        mobx.action(() => {
            this.isDone.set(true);
        })();
    }

    receiveData(pos: number, data: Uint8Array, reason?: string): void {
        this.packetData.receiveData(pos, data, reason);
    }

    updateHeight(newHeight: number): void {
        if (this.savedHeight != newHeight) {
            this.savedHeight = newHeight;
            this.api.saveHeight(newHeight);
        }
    }
}

@mobxReact.observer
class WaveAppRenderer extends React.Component<{ model: WaveAppRendererModel }, {}> {
    renderError() {
        const model = this.props.model;
        return <div className="load-error-text">{model.loadError.get()}</div>;
    }

    render() {
        let model = this.props.model;
        let styleVal = null;
        if (model.loading.get() && model.savedHeight >= 0 && model.isDone) {
            styleVal = {
                height: model.savedHeight,
                maxHeight: model.opts.maxSize.height,
            };
        } else {
            styleVal = {
                maxHeight: model.opts.maxSize.height,
            };
        }
        let version = model.version.get();
        let loadError = model.loadError.get();
        if (loadError != null) {
            return (
                <div className="waveapp-renderer" style={styleVal}>
                    {this.renderError()}
                </div>
            );
        }
        let node = model.data.get();
        if (node == null) {
            return <div className="waveapp-renderer" style={styleVal} />;
        }
        return (
            <div className="waveapp-renderer" style={styleVal}>
                {convertNodeToTag(node)}
            </div>
        );
    }
}

export { WaveAppRendererModel, WaveAppRenderer };
