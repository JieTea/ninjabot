package exchange

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
)

// AssetQuote 定义了交易对中的资产和报价货币。
type AssetQuote struct {
	Quote string
	Asset string
}

var (
	//go:embed pairs.json
	pairs             []byte                        //使用go:embed指令将pairs.json文件中的内容嵌入到编译后的二进制文件中。
	pairAssetQuoteMap = make(map[string]AssetQuote) //全局变量，用于存储交易对到资产/报价货币的映射关系。
)

// 包加载时执行，将pairs.json文件解析为映射表pairAssetQuoteMap
func init() {
	err := json.Unmarshal(pairs, &pairAssetQuoteMap)
	if err != nil {
		panic(err)
	}
}

// SplitAssetQuote 根据交易对获取对应的资产和报价货币
func SplitAssetQuote(pair string) (asset string, quote string) {
	data := pairAssetQuoteMap[pair]
	return data.Asset, data.Quote
}

// 通过 Binance API 更新交易对映射表，并将更新后的映射表保存到文件中。
// 这个函数会调用 Binance API 获取现货和期货交易对信息，
// 然后更新pairAssetQuoteMap并保存到文件中。
func updatePairsFile() error {
	client := binance.NewClient("", "")
	sportInfo, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get exchange info: %v", err)
	}

	futureClient := futures.NewClient("", "")
	futureInfo, err := futureClient.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get exchange info: %v", err)
	}

	for _, info := range sportInfo.Symbols {
		pairAssetQuoteMap[info.Symbol] = AssetQuote{
			Quote: info.QuoteAsset,
			Asset: info.BaseAsset,
		}
	}

	for _, info := range futureInfo.Symbols {
		pairAssetQuoteMap[info.Symbol] = AssetQuote{
			Quote: info.QuoteAsset,
			Asset: info.BaseAsset,
		}
	}

	fmt.Printf("Total pairs: %d\n", len(pairAssetQuoteMap))

	content, err := json.Marshal(pairAssetQuoteMap)
	if err != nil {
		return fmt.Errorf("failed to marshal pairs: %v", err)
	}

	err = os.WriteFile("pairs.json", content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	return nil
}
