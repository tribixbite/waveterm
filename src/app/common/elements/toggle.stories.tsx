import type { Meta, StoryObj } from "@storybook/react";

import { Toggle } from "./toggle";

//👇 This default export determines where your story goes in the story list
const meta: Meta<typeof Toggle> = {
    component: Toggle,
};

export default meta;
type Story = StoryObj<typeof Toggle>;

export const Basic: Story = {
    args: {
        checked: true,
        onChange: () => {},
    },
};
