#!/usr/bin/env node

const { exec } = require('child_process');
const path = require('path');

const SCRIPTS_DIR = path.join(__dirname, 'scripts');
const PYTHON_SCRIPT = path.join(SCRIPTS_DIR, 'fetch_stock.py');

async function executePython(args) {
    return new Promise((resolve, reject) => {
        const pythonCmd = process.env.PYTHON_CMD || 'python3';
        
        const cmd = exec(
            `${pythonCmd} ${PYTHON_SCRIPT} ${args.join(' ')}`,
            {
                cwd: SCRIPTS_DIR,
                maxBuffer: 10 * 1024 * 1024, // 10MB
            },
            (error, stdout, stderr) => {
                if (error) {
                    console.error(`执行失败: ${error.message}`);
                    console.error(`stderr: ${stderr}`);
                    reject({
                        success: false,
                        error: error.message,
                        stderr: stderr
                    });
                } else {
                    try {
                        const result = JSON.parse(stdout);
                        resolve(result);
                    } catch (e) {
                        // 如果输出不是JSON，返回原始文本
                        resolve({
                            success: true,
                            output: stdout.trim(),
                            stderr: stderr
                        });
                    }
                }
            }
        );
    });
}

async function analyzeStock(stock, market = 'auto') {
    console.log(`[INFO] 开始分析股票: ${stock} (市场: ${market})`);
    
    try {
        const args = [stock, '--market', market, '--output', 'json'];
        const result = await executePython(args);
        
        if (result.success) {
            const data = result.data || {};
            const name = data.name || stock;
            const price = data.price || '-';
            const change = data.change || '-';
            const changePercent = data.change_percent || '-';
            
            console.log(`[INFO] 成功获取数据: ${name}`);
            console.log(`[INFO] 价格: ${price}, 涨跌: ${change} (${changePercent})`);
            
            // 构建分析报告
            const report = generateReport(data);
            
            return {
                success: true,
                stock: stock,
                market: market,
                data: data,
                report: report,
                urls: data.urls || {}
            };
        } else {
            return {
                success: false,
                stock: stock,
                market: market,
                error: result.error || '获取数据失败',
                stderr: result.stderr
            };
        }
    } catch (error) {
        console.error(`[ERROR] 分析股票失败: ${error.message}`);
        return {
            success: false,
            stock: stock,
            market: market,
            error: error.message
        };
    }
}

function generateReport(data) {
    const name = data.name || '未知';
    const code = data.code || '-';
    const price = data.price || '-';
    const change = data.change || '-';
    const changePercent = data.change_percent || '-';
    const pe = data.pe || '-';
    const pb = data.pb || '-';
    const marketCap = data.market_cap || '-';
    const amount = data.amount || '-';
    
    // 判断涨跌
    let trend = '➡️';
    if (change !== '-' && parseFloat(change) > 0) {
        trend = '📈';
    } else if (change !== '-' && parseFloat(change) < 0) {
        trend = '📉';
    }
    
    // 基本面评分（简化版）
    let fundamentalScore = 5;
    let fundamentalComment = '基本面中性';
    
    if (pe !== '-' && parseFloat(pe) < 20) {
        fundamentalScore += 2;
        fundamentalComment = 'PE合理，估值较低';
    } else if (pe !== '-' && parseFloat(pe) > 50) {
        fundamentalScore -= 1;
        fundamentalComment = 'PE较高，估值偏高';
    }
    
    // 资金面评分（简化版）
    let fundScore = 5;
    let fundComment = '资金面中性';
    
    if (data.main_ratio) {
        const ratio = parseFloat(data.main_ratio);
        if (ratio > 10) {
            fundScore += 3;
            fundComment = '主力强势介入';
        } else if (ratio > 5) {
            fundScore += 2;
            fundComment = '主力温和流入';
        } else if (ratio < -5) {
            fundScore -= 2;
            fundComment = '主力明显流出';
        }
    }
    
    // 综合评分
    const totalScore = fundamentalScore + fundScore;
    let recommendation = '观望';
    let recommendationEmoji = '⚠️';
    
    if (totalScore >= 10) {
        recommendation = '推荐买入';
        recommendationEmoji = '✅';
    } else if (totalScore >= 8) {
        recommendation = '可以关注';
        recommendationEmoji = '👍';
    } else if (totalScore <= 4) {
        recommendation = '谨慎操作';
        recommendationEmoji = '❌';
    }
    
    // 买卖价位建议
    let buyPrice = '-';
    let targetPrice = '-';
    let stopPrice = '-';
    
    if (price !== '-') {
        const currentPrice = parseFloat(price);
        buyPrice = (currentPrice * 0.97).toFixed(2);
        targetPrice = (currentPrice * 1.1).toFixed(2);
        stopPrice = (currentPrice * 0.92).toFixed(2);
    }
    
    const report = `
═══════════════════════════════════════════════════════════════
  ${trend} ${name} (${code})
═════════════════════════════════════════════════════════════════

💰 当前价格: ${price}
${trend} 涨跌额:   ${change}
${trend} 涨跌幅:   ${changePercent}

────────────────────────────────────────────────────────────────────

📊 基本信息
  今开: ${data.open || '-'}
  昨收: ${data.prev_close || '-'}
  最高: ${data.high || '-'}
  最低: ${data.low || '-'}
  涨停: ${data.limit_up || '-'}
  跌停: ${data.limit_down || '-'}

────────────────────────────────────────────────────────────────────

💼 市场数据
  成交额: ${amount}
  总市值: ${marketCap}
  流通市值: ${data.float_cap || '-'}
  换手率: ${data.turnover || '-'}
  量比: ${data.volume_ratio || '-'}

────────────────────────────────────────────────────────────────────

📈 估值指标
  市盈率(PE): ${pe}
  市净率(PB): ${pb}

────────────────────────────────────────────────────────────────────

💡 综合分析
  基本面评分: ${fundamentalScore}/10
  ${fundamentalComment}
  
  资金面评分: ${fundScore}/10
  ${fundComment}
  
  综合评分: ${totalScore}/20
  ${recommendationEmoji} 投资建议: ${recommendation}

────────────────────────────────────────────────────────────────────

🎯 买卖价位建议
  建仓价位: ${buyPrice}
  目标价位: ${targetPrice}
  止损价位: ${stopPrice}

═════════════════════════════════════════════════════════════════
`;
    
    return report;
}

// 主程序入口
if (require.main === module) {
    const args = process.argv.slice(2);
    let command = 'analyze';
    let stock = '';
    let market = 'auto';
    
    // 解析命令行参数
    for (let i = 0; i < args.length; i++) {
        if (args[i] === '--command' && args[i + 1]) {
            command = args[i + 1];
        } else if (args[i] === '--stock' && args[i + 1]) {
            stock = args[i + 1];
        } else if (args[i] === '--market' && args[i + 1]) {
            market = args[i + 1];
        }
    }
    
    // 如果没有指定stock，从stdin读取
    if (!stock) {
        process.stdin.setEncoding('utf8');
        process.stdin.on('readable', () => {
            let chunk;
            while ((chunk = process.stdin.read()) !== null) {
                try {
                    const data = JSON.parse(chunk);
                    if (data.stock) stock = data.stock;
                    if (data.market) market = data.market;
                } catch (e) {
                    console.error(`[ERROR] JSON解析失败: ${e.message}`);
                }
            }
        });
        
        process.stdin.on('end', async () => {
            if (!stock) {
                console.error('[ERROR] 缺少必要参数: stock');
                console.error('Usage: node index.js --stock "股票代码" [--market "市场类型"]');
                process.exit(1);
            }
            
            const result = await analyzeStock(stock, market);
            console.log(JSON.stringify(result, null, 2));
            process.exit(result.success ? 0 : 1);
        });
    } else {
        analyzeStock(stock, market).then(result => {
            console.log(JSON.stringify(result, null, 2));
            process.exit(result.success ? 0 : 1);
        }).catch(error => {
            console.error(`[ERROR] ${error.message}`);
            process.exit(1);
        });
    }
}

module.exports = { analyzeStock, generateReport };
