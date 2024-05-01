import type { Meta, StoryObj } from "@storybook/react";

import { Button } from "./button";

//👇 This default export determines where your story goes in the story list
const meta: Meta<typeof Button> = {
    component: Button,
};

export default meta;
type Story = StoryObj<typeof Button>;

export const Button1: Story = {
    args: {
        children: "Click me",
        onClick: () => {},
    },
};

export const Button2: Story = {
    args: {
        ...Button1.args,
        children: "Click me too",
    },
};
