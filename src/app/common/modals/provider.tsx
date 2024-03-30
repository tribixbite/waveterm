// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React, { PureComponent } from "preact/compat";
import * as mobxReact from "mobx-preact";
import { GlobalModel } from "@/models";
import { TosModal } from "./tos";

@mobxReact.observer
class ModalsProvider extends PureComponent {
    render() {
        let store = GlobalModel.modalsModel.store.slice();
        if (GlobalModel.needsTos()) {
            return <TosModal />;
        }
        let rtn: JSX.Element[] = [];
        for (let i = 0; i < store.length; i++) {
            let entry = store[i];
            let Comp = entry.component;
            rtn.push(<Comp key={entry.uniqueKey} {...entry.props} />);
        }
        return <>{rtn}</>;
    }
}

export { ModalsProvider };
