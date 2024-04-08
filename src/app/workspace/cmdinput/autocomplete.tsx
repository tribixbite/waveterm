// Copyright 2024, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { SuggestionBlob } from "@/autocomplete/runtime/model";
import { GlobalModel } from "@/models";
import cn from "classnames";
import { useMemo } from "react";

export const AutocompleteSuggestions = async (props: { curLine: string }) => {
    const inputModel = GlobalModel.inputModel;
    const suggestions = await useMemo(async () => await inputModel.getSuggestions(), [props.curLine]);

    if (!suggestions) {
        return null;
    }

    const items = suggestions.suggestions.map((s) => `${s.icon} ${s.name}`);

    return (
        <div className="autocomplete-suggestions">
            {items.map((item, idx) => (
                <div key={idx} className={cn("autocomplete-item")}>
                    {item}
                </div>
            ))}
        </div>
    );
};
