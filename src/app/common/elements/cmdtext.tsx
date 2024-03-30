// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React, { PureComponent } from "preact/compat";

function renderCmdText(text: string): any {
    return <span>&#x2318;{text}</span>;
}

export { renderCmdText };
