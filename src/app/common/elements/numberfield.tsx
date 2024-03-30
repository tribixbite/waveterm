// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { JSX, cloneElement } from "preact";
import { boundMethod } from "autobind-decorator";

import { TextField } from "./textfield";

class NumberField extends TextField {
    @boundMethod
    handleInputChange(e: JSX.TargetedEvent) {
        const { required, onChange } = this.props;
        const inputValue = (e.target as HTMLInputElement).value;

        // Allow only numeric input
        if (inputValue === "" || /^\d*$/.test(inputValue)) {
            // Update the internal state only if the component is not controlled.
            if (this.props.value === undefined) {
                const isError = required ? inputValue.trim() === "" : false;

                this.setState({
                    internalValue: inputValue,
                    error: isError,
                    hasContent: Boolean(inputValue),
                });
            }

            onChange && onChange(inputValue);
        }
    }

    render() {
        // Use the render method from TextField but add the onKeyDown handler
        const renderedTextField = super.render();
        return cloneElement(renderedTextField);
    }
}

export { NumberField };
