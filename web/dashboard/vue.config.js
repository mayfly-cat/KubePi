const path = require('path')

function resolve(dir) {
    return path.join(__dirname, dir)
}

// 仪表盘应用部署在 /gvp/dashboard 之下，固定 publicPath，避免依赖环境变量
let publicPath = "/gvp/dashboard/"
module.exports = {
    outputDir: path.resolve(__dirname, '../../cmd/server/web/dashboard'),
    productionSourceMap: true,
    lintOnSave: false,
    devServer: {
        port: 4400,
        open: true,
        proxy: {
            '/gvp/api': {
                target: 'http://0.0.0.0:80',
                ws: true,
                secure: false,
            }
        }
    },
    configureWebpack: {
        performance: { 
            hints: false,
        },
        optimization: {
            minimize: true,
            splitChunks: {
                chunks: 'all',
            }
        },
        devtool: 'source-map',
        resolve: {
            alias: {
                '@': resolve('src')
            },
            fallback: {
                "path": false,
            }
        }
    },
    publicPath: `${publicPath}`
};
