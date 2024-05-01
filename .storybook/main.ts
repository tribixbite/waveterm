import type { StorybookConfig } from "@storybook/react-webpack5";
import { webProd, webDev } from "../webpack/webpack.web";
import { Configuration } from "webpack";

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
        let configToMerge = webDev;
        if (configType === "PRODUCTION") {
            configToMerge = webProd;
        }
        const ret: Configuration = {
            ...config,
            entry: configToMerge.entry,
            mode: configType === "PRODUCTION" ? "production" : "development",
            module: { ...config.module, rules: [...(config.module?.rules ?? []), ...configToMerge.module.rules] },
            resolve: {
                ...config.resolve,
                extensions: [...(config.resolve?.extensions ?? []), ...configToMerge.resolve.extensions],
                alias: { ...config.resolve?.alias, ...configToMerge.resolve.alias },
            },
            plugins: [...(config.plugins ?? []), ...configToMerge.plugins],
            devServer: configToMerge.devServer,
            devtool: configToMerge.devtool,
            output: {
                ...config.output,
                ...configToMerge.output,
            },
        };
        console.log(JSON.stringify(ret, null, 4));
        return ret;
    },
};
export default config;
