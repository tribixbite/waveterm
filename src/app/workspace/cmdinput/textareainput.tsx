// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import * as React from "react";
import * as mobx from "mobx";
import * as util from "@/util/util";
import { If } from "tsx-control-statements/components";
import cn from "classnames";
import { GlobalModel, GlobalCommandRunner, Screen } from "@/models";
import { getMonoFontSize } from "@/util/textmeasure";
import * as appconst from "@/app/appconst";
import { Observer } from "mobx-react";
import { SuggestionBlob } from "@/autocomplete/runtime/model";

type OV<T> = mobx.IObservableValue<T>;

function pageSize(div: any): number {
    if (div == null) {
        return 300;
    }
    let size = div.clientHeight;
    if (size > 500) {
        size = size - 100;
    } else if (size > 200) {
        size = size - 30;
    }
    return size;
}

function scrollDiv(div: any, amt: number) {
    if (div == null) {
        return;
    }
    let newScrollTop = div.scrollTop + amt;
    if (newScrollTop < 0) {
        newScrollTop = 0;
    }
    div.scrollTo({ top: newScrollTop, behavior: "smooth" });
}

const HistoryKeybindings = () => {
    React.useEffect(() => {
        if (GlobalModel.activeMainView != "session") {
            return;
        }
        const inputModel = GlobalModel.inputModel;
        const keybindManager = GlobalModel.keybindManager;
        keybindManager.registerKeybinding("pane", "history", "generic:cancel", (waveEvent) => {
            inputModel.resetHistory();
            return true;
        });
        keybindManager.registerKeybinding("pane", "history", "generic:confirm", (waveEvent) => {
            inputModel.grabSelectedHistoryItem();
            return true;
        });
        keybindManager.registerKeybinding("pane", "history", "history:closeHistory", (waveEvent) => {
            inputModel.resetInput();
            return true;
        });
        keybindManager.registerKeybinding("pane", "history", "history:toggleShowRemotes", (waveEvent) => {
            inputModel.toggleRemoteType();
            return true;
        });
        keybindManager.registerKeybinding("pane", "history", "history:changeScope", (waveEvent) => {
            inputModel.toggleHistoryType();
            return true;
        });
        keybindManager.registerKeybinding("pane", "history", "generic:selectAbove", (waveEvent) => {
            inputModel.moveHistorySelection(1);
            return true;
        });
        keybindManager.registerKeybinding("pane", "history", "generic:selectBelow", (waveEvent) => {
            inputModel.moveHistorySelection(-1);
            return true;
        });
        keybindManager.registerKeybinding("pane", "history", "generic:selectPageAbove", (waveEvent) => {
            inputModel.moveHistorySelection(10);
            return true;
        });
        keybindManager.registerKeybinding("pane", "history", "generic:selectPageBelow", (waveEvent) => {
            inputModel.moveHistorySelection(-10);
            return true;
        });
        keybindManager.registerKeybinding("pane", "history", "history:selectPreviousItem", (waveEvent) => {
            inputModel.moveHistorySelection(1);
            return true;
        });
        keybindManager.registerKeybinding("pane", "history", "history:selectNextItem", (waveEvent) => {
            inputModel.moveHistorySelection(-1);
            return true;
        });

        return () => {
            keybindManager.unregisterDomain("history");
        };
    });

    return null;
};

interface TextAreaInputCallbacks {
    arrowUpPressed: () => boolean;
    arrowDownPressed: () => boolean;
    scrollPage: (up: boolean) => void;
    modEnter: () => void;
    controlU: () => void;
    controlP: () => void;
    controlN: () => void;
    controlW: () => void;
    controlY: () => void;
    setLastHistoryUpDown: (lastHistoryUpDown: boolean) => void;
}

const CmdInputKeybindings = (props: { inputCallbacks: TextAreaInputCallbacks }) => {
    const [lastTab, setLastTab] = React.useState(false);
    const [curPress, setCurPress] = React.useState("");

    React.useEffect(() => {
        if (GlobalModel.activeMainView != "session") {
            return;
        }
        const inputCallbacks = props.inputCallbacks;
        setLastTab(false);
        const keybindManager = GlobalModel.keybindManager;
        const inputModel = GlobalModel.inputModel;
        keybindManager.registerKeybinding("pane", "cmdinput", "cmdinput:autocomplete", (waveEvent) => {
            const lastTab_ = lastTab;
            setLastTab(true);
            setCurPress("tab");
            const curLine = inputModel.getCurLine();
            // if (lastTab) {
            inputModel.getSuggestions().then(
                mobx.action((resp) => {
                    console.log("resp", resp);
                    inputModel.flashInfoMsg({ infotitle: resp?.suggestions[0].name }, 10000);
                })
            );
            // } else {
            //     GlobalModel.submitCommand(
            //         "_compgen",
            //         null,
            //         [curLine],
            //         { comppos: String(curLine.length), nohist: "1" },
            //         true
            //     );
            // }
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "generic:confirm", (waveEvent) => {
            GlobalModel.closeTabSettings();
            if (GlobalModel.inputModel.isEmpty()) {
                const activeWindow = GlobalModel.getScreenLinesForActiveScreen();
                const activeScreen = GlobalModel.getActiveScreen();
                if (activeScreen != null && activeWindow != null && activeWindow.lines.length > 0) {
                    activeScreen.setSelectedLine(0);
                    GlobalCommandRunner.screenSelectLine("E");
                }
            } else {
                setTimeout(() => GlobalModel.inputModel.uiSubmitCommand(), 0);
            }
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "generic:cancel", (waveEvent) => {
            GlobalModel.closeTabSettings();
            inputModel.closeAuxView();
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "cmdinput:expandInput", (waveEvent) => {
            inputModel.toggleExpandInput();
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "cmdinput:clearInput", (waveEvent) => {
            inputModel.resetInput();
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "cmdinput:cutLineLeftOfCursor", (waveEvent) => {
            inputCallbacks.controlU();
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "cmdinput:cutWordLeftOfCursor", (waveEvent) => {
            inputCallbacks.controlW();
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "cmdinput:paste", (waveEvent) => {
            inputCallbacks.controlY();
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "cmdinput:openHistory", (waveEvent) => {
            inputModel.openHistory();
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "cmdinput:previousHistoryItem", (waveEvent) => {
            setCurPress("historyupdown");
            inputCallbacks.controlP();
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "cmdinput:nextHistoryItem", (waveEvent) => {
            setCurPress("historyupdown");
            inputCallbacks.controlN();
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "cmdinput:openAIChat", (waveEvent) => {
            inputModel.openAIAssistantChat();
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "generic:selectAbove", (waveEvent) => {
            setCurPress("historyupdown");
            const rtn = inputCallbacks.arrowUpPressed();
            return rtn;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "generic:selectBelow", (waveEvent) => {
            setCurPress("historyupdown");
            const rtn = inputCallbacks.arrowDownPressed();
            return rtn;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "generic:selectPageAbove", (waveEvent) => {
            setCurPress("historyupdown");
            inputCallbacks.scrollPage(true);
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "generic:selectPageBelow", (waveEvent) => {
            setCurPress("historyupdown");
            inputCallbacks.scrollPage(false);
            return true;
        });
        keybindManager.registerKeybinding("pane", "cmdinput", "generic:expandTextInput", (waveEvent) => {
            inputCallbacks.modEnter();
            return true;
        });
        keybindManager.registerDomainCallback("cmdinput", (waveEvent) => {
            if (curPress != "tab") {
                setLastTab(false);
            }
            if (curPress != "historyupdown") {
                inputCallbacks.setLastHistoryUpDown(false);
            }
            setCurPress("");
            return false;
        });

        return () => {
            keybindManager.unregisterDomain("cmdinput");
        };
    });

    return null;
};

const TextAreaInput = (props: { screen: Screen; onHeightChange: () => void }) => {
    const [lastHistoryUpDown, setLastHistoryUpDown] = React.useState(false);
    const [lastHeight, setLastHeight] = React.useState(0);
    const [lastSP, setLastSP] = React.useState({ str: "", pos: appconst.NoStrPos });
    const mainInputRef: React.RefObject<HTMLDivElement> = React.useRef();
    const historyInputRef: React.RefObject<HTMLInputElement> = React.useRef();
    const controlRef: React.RefObject<HTMLDivElement> = React.useRef();
    const activeScreen = GlobalModel.getActiveScreen();
    // Local state for the current input value
    const [curInput, setCurInput] = React.useState(GlobalModel.inputModel.getCurLine());
    const [suggestions, setSuggestions] = React.useState<SuggestionBlob>(null);
    const inputModel = GlobalModel.inputModel;

    const getSelection = (elem: HTMLElement): { text: string; startPos: number; endPos: number } => {
        if (elem == null) {
            return { text: "", startPos: 0, endPos: 0 };
        }
        const text = elem.textContent;
        const selection = window.getSelection();
        if (selection == null || selection.anchorNode != elem) {
            return { text: "", startPos: 0, endPos: 0 };
        }
        const startPos = selection.anchorOffset;
        const endPos = selection.focusOffset;
        return { text, startPos, endPos };
    };

    const getCurSP = (): StrWithPos => {
        const { text, startPos, endPos } = getSelection(mainInputRef.current);
        if (text == null || startPos != endPos) {
            return { str: "", pos: appconst.NoStrPos };
        }
        return { str: text, pos: startPos };
    };

    const updateSP = () => {
        const curSP = getCurSP();
        if (curSP.str == lastSP.str && curSP.pos == lastSP.pos) {
            return;
        }
        setLastSP(curSP);
        GlobalModel.sendCmdInputText(props.screen.screenId, curSP);
    };

    const setFocus = () => {
        GlobalModel.inputModel.giveFocus();
    };

    const getTextAreaMaxCols = (): number => {
        const taElem = mainInputRef.current;
        if (taElem == null) {
            return 0;
        }
        const cs = window.getComputedStyle(taElem);
        const padding = parseFloat(cs.paddingLeft) + parseFloat(cs.paddingRight);
        const borders = parseFloat(cs.borderLeft) + parseFloat(cs.borderRight);
        const contentWidth = taElem.clientWidth - padding - borders;
        const fontSize = getMonoFontSize(parseInt(cs.fontSize));
        const maxCols = Math.floor(contentWidth / Math.ceil(fontSize.width));
        return maxCols;
    };

    const checkHeight = (shouldFire: boolean) => {
        const elem = controlRef.current;
        if (elem == null) {
            return;
        }
        const curHeight = elem.offsetHeight;
        if (lastHeight == curHeight) {
            return;
        }
        setLastHeight(curHeight);
        if (shouldFire && props.onHeightChange != null) {
            props.onHeightChange();
        }
    };

    // Called whenever the component is mounted or updated
    React.useEffect(
        () =>
            mobx.autorun(() => {
                console.log("TextAreaInput useEffect");

                // Set the local state to the current input value
                setCurInput(GlobalModel.inputModel.getCurLine());
                if (activeScreen != null) {
                    const focusType: FocusTypeStrs = activeScreen.focusType.get();
                    if (focusType == "input") {
                        setFocus();
                    }
                }
                checkHeight(false);
                updateSP();
            }),
        []
    );

    // Sets the focus on the input element when the forceInputFocus flag is set
    React.useEffect(() => {
        if (inputModel.forceInputFocus) {
            inputModel.forceInputFocus = false;
            setFocus();
        }
    }, [inputModel.forceInputFocus]);

    // Called whenever the input cursor position is forced to change
    React.useEffect(
        () =>
            mobx.autorun(() => {
                const inputModel = GlobalModel.inputModel;
                const fcpos = inputModel.forceCursorPos.get();
                if (fcpos != null && fcpos != appconst.NoStrPos) {
                    if (mainInputRef.current != null) {
                        const selectRange = document.createRange();
                        selectRange.setStart(mainInputRef.current, fcpos);
                        selectRange.setEnd(mainInputRef.current, fcpos);
                        const sel = window.getSelection();
                        sel.removeAllRanges();
                        sel.addRange(selectRange);
                    }
                    mobx.action(() => inputModel.forceCursorPos.set(null))();
                }
            }),
        []
    );

    // Asynchronously load autocomplete suggestions when the input changes
    React.useEffect(() => {
        let active = true;

        const loadSuggestions = async () => {
            const inputModel = GlobalModel.inputModel;
            const suggestions = await inputModel.getSuggestions();
            if (!active) {
                return;
            }
            setSuggestions(suggestions);
        };

        loadSuggestions();
        return () => {
            active = false;
        };
    }, [curInput]);

    const getLinePos = (elem: any): { numLines: number; linePos: number } => {
        const numLines = elem.value.split("\n").length;
        const linePos = elem.value.substr(0, elem.selectionStart).split("\n").length;
        return { numLines, linePos };
    };

    const onChange = (e: any) => {
        mobx.action(() => {
            GlobalModel.inputModel.setCurLine(e.target.value);
            setCurInput(e.target.value);
        })();
    };

    const handleHistoryInput = (e: any) => {
        const inputModel = GlobalModel.inputModel;
        mobx.action(() => {
            const opts = mobx.toJS(inputModel.historyQueryOpts.get());
            opts.queryStr = e.target.value;
            inputModel.setHistoryQueryOpts(opts);
        })();
    };

    const handleFocus = (e: any) => {
        e.preventDefault();
        GlobalModel.inputModel.giveFocus();
    };

    const handleMainBlur = (e: any) => {
        if (document.activeElement == mainInputRef.current) {
            return;
        }
        GlobalModel.inputModel.setPhysicalInputFocused(false);
    };

    const handleHistoryBlur = (e: any) => {
        if (document.activeElement == historyInputRef.current) {
            return;
        }
        GlobalModel.inputModel.setPhysicalInputFocused(false);
    };

    let displayLines = 1;

    const numLines = curInput.split("\n").length;
    const maxCols = getTextAreaMaxCols();
    let longLine = false;
    if (maxCols != 0 && curInput.length >= maxCols - 4) {
        longLine = true;
    }
    if (numLines > 1 || longLine || inputModel.inputExpanded.get()) {
        displayLines = 5;
    }

    const auxViewFocused = inputModel.getAuxViewFocus();
    if (auxViewFocused) {
        displayLines = 1;
    }
    if (activeScreen != null) {
        activeScreen.focusType.get(); // for reaction
    }
    const termFontSize = GlobalModel.getTermFontSize();
    const fontSize = getMonoFontSize(termFontSize);
    const termPad = fontSize.pad;
    const computedInnerHeight = displayLines * fontSize.height + 2 * termPad;
    const computedOuterHeight = computedInnerHeight + 2 * termPad;
    let shellType: string = "";
    if (screen != null) {
        const ri = activeScreen.getCurRemoteInstance();
        if (ri?.shelltype != null) {
            shellType = ri.shelltype;
        }
        if (shellType == "") {
            const rptr = activeScreen.curRemote.get();
            if (rptr != null) {
                const remote = GlobalModel.getRemote(rptr.remoteid);
                if (remote != null) {
                    shellType = remote.defaultshelltype;
                }
            }
        }
    }
    const isHistoryFocused = auxViewFocused && inputModel.getActiveAuxView() == appconst.InputAuxView_History;

    const keybindingCallbacks: TextAreaInputCallbacks = {
        arrowUpPressed: (): boolean => {
            const inputModel = GlobalModel.inputModel;
            if (!inputModel.isHistoryLoaded()) {
                setLastHistoryUpDown(true);
                inputModel.loadHistory(false, 1, "screen");
                return true;
            }
            const currentRef = mainInputRef.current;
            if (currentRef == null) {
                return true;
            }
            const linePos = getLinePos(currentRef);
            const lastHist = lastHistoryUpDown;
            if (!lastHist && linePos.linePos > 1) {
                // regular arrow
                return false;
            }
            inputModel.moveHistorySelection(1);
            setLastHistoryUpDown(true);
            return true;
        },
        arrowDownPressed: (): boolean => {
            const inputModel = GlobalModel.inputModel;
            if (!inputModel.isHistoryLoaded()) {
                return true;
            }
            const currentRef = mainInputRef.current;
            if (currentRef == null) {
                return true;
            }
            const linePos = getLinePos(currentRef);
            const lastHist = lastHistoryUpDown;
            if (!lastHist && linePos.linePos < linePos.numLines) {
                // regular arrow
                return false;
            }
            inputModel.moveHistorySelection(-1);
            setLastHistoryUpDown(true);
            return true;
        },
        scrollPage: (up: boolean) => {
            const inputModel = GlobalModel.inputModel;
            const infoScroll = inputModel.hasScrollingInfoMsg();
            if (infoScroll) {
                const div = document.querySelector(".cmd-input-info");
                const amt = pageSize(div);
                scrollDiv(div, up ? -amt : amt);
            }
        },
        modEnter: () => {
            const currentRef = mainInputRef.current;
            if (currentRef == null) {
                return;
            }
            GlobalModel.inputModel.setCurLine(currentRef.textContent);
        },
        controlU: () => {
            if (mainInputRef.current == null) {
                return;
            }
            const { text, startPos, endPos } = getSelection(mainInputRef.current);
            if (startPos > text.length) {
                return;
            }
            const cutValue = text.substring(0, startPos);
            const restValue = text.substring(startPos);
            const cmdLineUpdate = { str: restValue, pos: 0 };
            navigator.clipboard.writeText(cutValue);
            GlobalModel.inputModel.updateCmdLine(cmdLineUpdate);
        },
        controlP: () => {
            const inputModel = GlobalModel.inputModel;
            if (!inputModel.isHistoryLoaded()) {
                setLastHistoryUpDown(true);
                inputModel.loadHistory(false, 1, "screen");
                return;
            }
            inputModel.moveHistorySelection(1);
            setLastHistoryUpDown(true);
        },
        controlN: () => {
            const inputModel = GlobalModel.inputModel;
            inputModel.moveHistorySelection(-1);
            setLastHistoryUpDown(true);
        },
        controlW: () => {
            if (mainInputRef.current == null) {
                return;
            }
            const { text, startPos, endPos } = getSelection(mainInputRef.current);
            if (startPos > text.length) {
                return;
            }
            let cutSpot = startPos - 1;
            let initial = true;
            for (; cutSpot >= 0; cutSpot--) {
                const ch = text[cutSpot];
                if (ch == " " && initial) {
                    continue;
                }
                initial = false;
                if (ch == " ") {
                    cutSpot++;
                    break;
                }
            }
            if (cutSpot == -1) {
                cutSpot = 0;
            }
            const cutValue = text.slice(cutSpot, startPos);
            const prevValue = text.slice(0, cutSpot);
            const restValue = text.slice(startPos);
            const cmdLineUpdate = { str: prevValue + restValue, pos: prevValue.length };
            navigator.clipboard.writeText(cutValue);
            GlobalModel.inputModel.updateCmdLine(cmdLineUpdate);
        },
        controlY: () => {
            if (mainInputRef.current == null) {
                return;
            }
            const pastePromise = navigator.clipboard.readText();
            pastePromise.then((clipText) => {
                clipText = clipText ?? "";
                const { text, startPos, endPos } = getSelection(mainInputRef.current);
                const selStart = startPos;
                const selEnd = endPos;
                const value = text;
                if (selStart > value.length || selEnd > value.length) {
                    return;
                }
                const newValue = value.substring(0, selStart) + clipText + value.substring(selEnd);
                const cmdLineUpdate = { str: newValue, pos: selStart + clipText.length };
                GlobalModel.inputModel.updateCmdLine(cmdLineUpdate);
            });
        },
        setLastHistoryUpDown,
    };
    return (
        <Observer>
            {() => {
                return (
                    <div
                        className="textareainput-div control is-expanded"
                        ref={controlRef}
                        style={{ height: computedOuterHeight }}
                    >
                        <If condition={!auxViewFocused}>
                            <CmdInputKeybindings inputCallbacks={keybindingCallbacks} />
                        </If>
                        <If condition={isHistoryFocused}>
                            <HistoryKeybindings />
                        </If>

                        <If condition={!util.isBlank(shellType)}>
                            <div className="shelltag">{shellType}</div>
                        </If>
                        <div
                            contentEditable={true}
                            key="main"
                            ref={mainInputRef}
                            spellCheck="false"
                            autoCorrect="off"
                            id="main-cmd-input"
                            onFocus={handleFocus}
                            onBlur={handleMainBlur}
                            onChange={onChange}
                            className={cn("textarea", { "display-disabled": auxViewFocused })}
                            style={{
                                height: computedInnerHeight,
                                minHeight: computedInnerHeight,
                                fontSize: termFontSize,
                            }}
                        >
                            {curInput}
                        </div>
                        <If condition={suggestions != null && suggestions.suggestions.length > 0}>
                            <span
                                className="suggestion"
                                style={{
                                    height: computedInnerHeight,
                                    minHeight: computedInnerHeight,
                                    fontSize: termFontSize,
                                }}
                            >
                                {suggestions.suggestions[0].name}
                            </span>
                        </If>
                        <input
                            key="history"
                            ref={historyInputRef}
                            spellCheck="false"
                            autoComplete="off"
                            autoCorrect="off"
                            className="history-input"
                            type="text"
                            onFocus={handleFocus}
                            onBlur={handleHistoryBlur}
                            onChange={handleHistoryInput}
                            value={inputModel.historyQueryOpts.get().queryStr}
                        />
                    </div>
                );
            }}
        </Observer>
    );
};

export { TextAreaInput };
