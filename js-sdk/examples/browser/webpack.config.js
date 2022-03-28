const path = require("path");
const HtmlWebpackPlugin = require("html-webpack-plugin");

module.exports = {
  entry: path.resolve(__dirname, "./browser-src/index.ts"),
  devtool: 'inline-source-map',
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        exclude: /node_modules/,
        use: ["ts-loader"],
      },
    ],
  },
  resolve: {
    extensions: [".tsx", ".ts", ".js"],
    modules: [path.resolve(__dirname, "./node_modules"), "node_modules"],
  },
  output: {
    path: path.resolve(__dirname, "./browser-dist"),
    filename: "bundle.js",
  },
  devServer: {
    contentBase: path.resolve(__dirname, "./browser-dist"),
  },
  // This will output an HTML file that embeds the minified JS file
  // Put both in the same directory when hosting.
  plugins: [
    new HtmlWebpackPlugin({
      template: "browser-src/index.html",
      filename: "opaque-demo.html",
    }),
  ],
};
