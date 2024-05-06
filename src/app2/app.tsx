// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import * as React from "react";
import * as mobxReact from "mobx-react";
import * as mobx from "mobx";
import { clsx } from "clsx";

import { If } from "tsx-control-statements/components";
import { GlobalModel } from "@/models";
import { isBlank } from "@/util/util";
import { WorkspaceView } from "@/app/workspace/workspaceview";
import { PluginsView } from "@/app/pluginsview/pluginsview";
import { BookmarksView } from "@/app/bookmarks/bookmarks";
import { HistoryView } from "@/app/history/history";
import { ConnectionsView } from "@/app/connections/connections";
import { ClientSettingsView } from "@/app/clientsettings/clientsettings";
import { MainSideBar } from "@/app/sidebar/main";
import { RightSideBar } from "@/app/sidebar/right";
import { DisconnectedModal, ClientStopModal } from "@/modals";
import { ModalsProvider } from "@/modals/provider";
import { Button, TermStyleList } from "@/elements";
import { ErrorBoundary } from "@/common/error/errorboundary";

import "@/app/app.less";

export const App2: React.FC = mobxReact.observer(() => {
    const [dcWait, setDcWait] = React.useState(false);
    const mainContentRef = React.useRef<HTMLDivElement>(null);
    const [termThemesLoaded, setTermThemesLoaded] = React.useState(false);

    const handleContextMenu = (e: React.MouseEvent<HTMLElement>) => {
        let isInNonTermInput = false;
        const activeElem = document.activeElement;
        if (activeElem != null && activeElem.nodeName == "TEXTAREA") {
            if (!activeElem.classList.contains("xterm-helper-textarea")) {
                isInNonTermInput = true;
            }
        }
        if (activeElem != null && activeElem.nodeName == "INPUT" && activeElem.getAttribute("type") == "text") {
            isInNonTermInput = true;
        }
        const opts: ContextMenuOpts = {};
        if (isInNonTermInput) {
            opts.showCut = true;
        }
        const sel = window.getSelection();
        if (!isBlank(sel?.toString()) || isInNonTermInput) {
            GlobalModel.contextEditMenu(e, opts);
        }
    };

    const openMainSidebar = mobx.action(() => {
        const mainSidebarModel = GlobalModel.mainSidebarModel;
        const width = mainSidebarModel.getWidth(true);
        mainSidebarModel.saveState(width, false);
    });

    const openRightSidebar = mobx.action(() => {
        const rightSidebarModel = GlobalModel.rightSidebarModel;
        const width = rightSidebarModel.getWidth(true);
        rightSidebarModel.saveState(width, false);
    });

    const remotesModel = GlobalModel.remotesModel;
    const disconnected = !GlobalModel.ws.open.get() || !GlobalModel.waveSrvRunning.get();
    const hasClientStop = GlobalModel.getHasClientStop();
    const platform = GlobalModel.getPlatform();
    const clientData = GlobalModel.clientData.get();

    // Previously, this is done in sidebar.tsx but it causes flicker when clientData is null cos screen-view shifts around.
    // Doing it here fixes the flicker cos app is not rendered until clientData is populated.
    // wait for termThemes as well (this actually means that the "connect" packet has been received)
    if (clientData == null || GlobalModel.termThemes.get() == null) {
        return null;
    }

    if (disconnected || hasClientStop) {
        if (!dcWait) {
            setTimeout(() => setDcWait(true), 1500);
        }
        return (
            <div id="main" className={"platform-" + platform} onContextMenu={handleContextMenu}>
                <div ref={mainContentRef} className="main-content">
                    <MainSideBar parentRef={mainContentRef} />
                    <div className="session-view" />
                </div>
                <If condition={dcWait}>
                    <If condition={disconnected}>
                        <DisconnectedModal />
                    </If>
                    <If condition={!disconnected && hasClientStop}>
                        <ClientStopModal />
                    </If>
                </If>
            </div>
        );
    }

    if (dcWait) {
        setTimeout(() => setDcWait(false), 0);
    }

    // used to force a full reload of the application
    const renderVersion = GlobalModel.renderVersion.get();
    const mainSidebarCollapsed = GlobalModel.mainSidebarModel.getCollapsed();
    const rightSidebarCollapsed = GlobalModel.rightSidebarModel.getCollapsed();
    const activeMainView = GlobalModel.activeMainView.get();
    const lightDarkClass = GlobalModel.isDarkTheme.get() ? "is-dark" : "is-light";
    const mainClassName = clsx(
        "platform-" + platform,
        {
            "mainsidebar-collapsed": mainSidebarCollapsed,
            "rightsidebar-collapsed": rightSidebarCollapsed,
        },
        lightDarkClass
    );
    return (
        <>
            <TermStyleList onRendered={() => setTermThemesLoaded(true)} />
            <div
                key={`version- + ${renderVersion}`}
                id="main"
                className={mainClassName}
                onContextMenu={handleContextMenu}
            >
                <If condition={termThemesLoaded}>
                    <If condition={mainSidebarCollapsed}>
                        <div key="logo-button" className="logo-button-container">
                            <div className="logo-button-spacer" />
                            <div className="logo-button" onClick={openMainSidebar}>
                                <img src="public/logos/wave-logo.png" alt="logo" />
                            </div>
                        </div>
                    </If>
                    <If condition={GlobalModel.isDev && rightSidebarCollapsed && activeMainView == "session"}>
                        <div className="right-sidebar-triggers">
                            <Button className="secondary ghost right-sidebar-trigger" onClick={openRightSidebar}>
                                <i className="fa-sharp fa-solid fa-sidebar-flip"></i>
                            </Button>
                        </div>
                    </If>
                    <div ref={mainContentRef} className="main-content">
                        <MainSideBar parentRef={mainContentRef} />
                        <ErrorBoundary>
                            <PluginsView />
                            <WorkspaceView />
                            <HistoryView />
                            <BookmarksView />
                            <ConnectionsView model={remotesModel} />
                            <ClientSettingsView model={remotesModel} />
                        </ErrorBoundary>
                        <RightSideBar parentRef={mainContentRef} />
                    </div>
                    <ModalsProvider />
                </If>
            </div>
        </>
    );
});
