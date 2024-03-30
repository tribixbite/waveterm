import React, { PureComponent, ReactNode, CSSProperties } from "preact/compat";
import { boundMethod } from "autobind-decorator";
import cn from "classnames";

import "./button.less";

interface ButtonProps {
    children: ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    leftIcon?: ReactNode;
    rightIcon?: ReactNode;
    style?: CSSProperties;
    autoFocus?: boolean;
    className?: string;
    termInline?: boolean;
}

class Button extends PureComponent<ButtonProps> {
    static defaultProps = {
        style: {},
        className: "primary",
    };

    @boundMethod
    handleClick() {
        if (this.props.onClick && !this.props.disabled) {
            this.props.onClick();
        }
    }

    render() {
        const { leftIcon, rightIcon, children, disabled, style, autoFocus, termInline, className } = this.props;

        return (
            <button
                className={cn("wave-button", { disabled }, { "term-inline": termInline }, className)}
                onClick={this.handleClick}
                disabled={disabled}
                style={style}
                autoFocus={autoFocus}
            >
                {leftIcon && <span className="icon-left">{leftIcon}</span>}
                {children}
                {rightIcon && <span className="icon-right">{rightIcon}</span>}
            </button>
        );
    }
}

export { Button };
export type { ButtonProps };
