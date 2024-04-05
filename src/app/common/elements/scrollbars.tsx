import { OverlayScrollbarsComponentProps, useOverlayScrollbars } from "overlayscrollbars-react";
import React from "react";

export const ScrollbarsComponent = (props: {
    options?: OverlayScrollbarsComponentProps["options"];
    children: React.ReactNode;
    childrenRef: React.RefObject<any>;
}) => {
    const [initialize, instance] = useOverlayScrollbars({ options: props.options });

    React.useEffect(() => {
        initialize(props.childrenRef.current);
    }, [initialize, props.childrenRef.current]);

    return <div ref={props.childrenRef}>{props.children}</div>;
};
