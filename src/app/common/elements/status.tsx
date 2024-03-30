// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React, { PureComponent } from "preact/compat";
import { boundMethod } from "autobind-decorator";

import "./status.less";

interface StatusProps {
    status: "green" | "red" | "gray" | "yellow";
    text: string;
}

class Status extends PureComponent<StatusProps> {
    @boundMethod
    renderDot() {
        const { status } = this.props;

        return <div className={`dot ${status}`} />;
    }

    render() {
        const { text } = this.props;

        return (
            <div className="wave-status-container">
                {this.renderDot()}
                <span>{text}</span>
            </div>
        );
    }
}

export { Status };
