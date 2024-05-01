import type { StorybookConfig } from "@storybook/react-webpack5";
import { webProd, webDev } from "../webpack/webpack.web";
import { electronProd, electronDev } from "../webpack/webpack.electron";
const webpackMerge = require("webpack-merge");

const config: StorybookConfig = {
    stories: ["../src/**/*.mdx", "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)"],
    addons: [
        "@storybook/addon-onboarding",
        "@storybook/addon-links",
        "@storybook/addon-essentials",
        "@chromatic-com/storybook",
        "@storybook/addon-interactions",
        "@storybook/addon-webpack5-compiler-swc",
        "@storybook/addon-styling-webpack",
    ],
    framework: {
        name: "@storybook/react-webpack5",
        options: {},
    },
    docs: {
        autodocs: "tag",
    },
    webpackFinal: (config, { configType }) => {
        let configToMerge = electronDev;
        if (configType === "PRODUCTION") {
            configToMerge = electronProd;
        }
        const ret = webpackMerge.merge(config, configToMerge);
        console.log("config", ret);
        return ret;
    },
};
export default config;
