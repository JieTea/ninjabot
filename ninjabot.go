package ninjabot

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aybabtme/uniplot/histogram"

	"github.com/rodrigo-brito/ninjabot/exchange"
	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/notification"
	"github.com/rodrigo-brito/ninjabot/order"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/storage"
	"github.com/rodrigo-brito/ninjabot/strategy"
	"github.com/rodrigo-brito/ninjabot/tools/log"
	"github.com/rodrigo-brito/ninjabot/tools/metrics"

	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
)

const defaultDatabase = "ninjabot.db"

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04",
	})
}

type OrderSubscriber interface {
	OnOrder(model.Order)
}

type CandleSubscriber interface {
	OnCandle(model.Candle)
}

type NinjaBot struct {
	storage  storage.Storage   // 持久化存储的接口
	settings model.Settings    // 配置信息
	exchange service.Exchange  // 交易所的接口，用于与交易所进行交互
	strategy strategy.Strategy // 交易策略，定义了机器人的交易逻辑
	notifier service.Notifier  // 通知器，用于发送通知消息
	telegram service.Telegram  // 用于通过电报发送消息

	orderController       *order.Controller               // 订单控制器，用于管理订单的创建、取消等操作。
	priorityQueueCandle   *model.PriorityQueue            // 优先级队列，用于按照时间顺序处理K线数据。
	strategiesControllers map[string]*strategy.Controller // 策略控制器，存储不同交易对对应的策略控制器。
	orderFeed             *order.Feed                     // 订单数据源，用于接收订单信息
	dataFeed              *exchange.DataFeedSubscription  // 数据源订阅，用于订阅交易数据。
	paperWallet           *exchange.PaperWallet           // Paper钱包，用于模拟交易

	backtest bool // 是否在回测模式下运行
}

type Option func(*NinjaBot)

func NewBot(ctx context.Context, settings model.Settings, exch service.Exchange, str strategy.Strategy,
	options ...Option) (*NinjaBot, error) {

	bot := &NinjaBot{
		settings:              settings,
		exchange:              exch,
		strategy:              str,
		orderFeed:             order.NewOrderFeed(),
		dataFeed:              exchange.NewDataFeed(exch),
		strategiesControllers: make(map[string]*strategy.Controller),
		priorityQueueCandle:   model.NewPriorityQueue(nil),
	}

	for _, pair := range settings.Pairs {
		asset, quote := exchange.SplitAssetQuote(pair)
		if asset == "" || quote == "" {
			return nil, fmt.Errorf("invalid pair: %s", pair)
		}
	}

	for _, option := range options {
		option(bot)
	}

	var err error
	if bot.storage == nil {
		bot.storage, err = storage.FromFile(defaultDatabase)
		if err != nil {
			return nil, err
		}
	}

	bot.orderController = order.NewController(ctx, exch, bot.storage, bot.orderFeed)

	if settings.Telegram.Enabled {
		bot.telegram, err = notification.NewTelegram(bot.orderController, settings)
		if err != nil {
			return nil, err
		}
		// register telegram as notifier
		WithNotifier(bot.telegram)(bot)
	}

	return bot, nil
}

// WithBacktest 将机器人设置为运行在回测模式下，这在回测环境中是必需的
// 回测模式优化了用于 CSV 的输入读取，并处理了竞态条件
// WithBacktest sets the bot to run in backtest mode, it is required for backtesting environments
// Backtest mode optimize the input read for CSV and deal with race conditions
func WithBacktest(wallet *exchange.PaperWallet) Option {
	return func(bot *NinjaBot) {
		bot.backtest = true
		opt := WithPaperWallet(wallet)
		opt(bot)
	}
}

// WithStorage 设置机器人的存储， 默认使用一个名为 ninjabot.db 的本地文件
// WithStorage sets the storage for the bot, by default it uses a local file called ninjabot.db
func WithStorage(storage storage.Storage) Option {
	return func(bot *NinjaBot) {
		bot.storage = storage
	}
}

// WithLogLevel 设置日志级别。例如: log.DebugLevel、log.InfoLevel、log.WarnLevel、log.ErrorLevel、log.FatalLevel
// WithLogLevel sets the log level. eg: log.DebugLevel, log.InfoLevel, log.WarnLevel, log.ErrorLevel, log.FatalLevel
func WithLogLevel(level log.Level) Option {
	return func(_ *NinjaBot) {
		log.SetLevel(level)
	}
}

// WithNotifier 向机器人注册一个通知器，当前仅支持电子邮件和电报
// WithNotifier registers a notifier to the bot, currently only email and telegram are supported
func WithNotifier(notifier service.Notifier) Option {
	return func(bot *NinjaBot) {
		bot.notifier = notifier
		bot.orderController.SetNotifier(notifier)
		bot.SubscribeOrder(notifier)
	}
}

// WithCandleSubscription 将给定的结构订阅到蜡烛图数据流中
// WithCandleSubscription subscribes a given struct to the candle feed
func WithCandleSubscription(subscriber CandleSubscriber) Option {
	return func(bot *NinjaBot) {
		bot.SubscribeCandle(subscriber)
	}
}

// WithPaperWallet 为机器人设置纸钱包（用于回测和实时模拟）
// WithPaperWallet sets the paper wallet for the bot (used for backtesting and live simulation)
func WithPaperWallet(wallet *exchange.PaperWallet) Option {
	return func(bot *NinjaBot) {
		bot.paperWallet = wallet
	}
}

func (n *NinjaBot) SubscribeCandle(subscriptions ...CandleSubscriber) {
	for _, pair := range n.settings.Pairs {
		for _, subscription := range subscriptions {
			n.dataFeed.Subscribe(pair, n.strategy.Timeframe(), subscription.OnCandle, false)
		}
	}
}

// WithOrderSubscription 将给定的结构订阅到订单数据流中
func WithOrderSubscription(subscriber OrderSubscriber) Option {
	return func(bot *NinjaBot) {
		bot.SubscribeOrder(subscriber)
	}
}

func (n *NinjaBot) SubscribeOrder(subscriptions ...OrderSubscriber) {
	for _, pair := range n.settings.Pairs {
		for _, subscription := range subscriptions {
			n.orderFeed.Subscribe(pair, subscription.OnOrder, false)
		}
	}
}

func (n *NinjaBot) Controller() *order.Controller {
	return n.orderController
}

// Summary 函数在 stdout 中显示所有交易、准确度和一些机器人指标
// 要访问原始数据，可以访问 `bot.Controller().Results`
// Summary function displays all trades, accuracy and some bot metrics in stdout
// To access the raw data, you may access `bot.Controller().Results`
func (n *NinjaBot) Summary() {
	var (
		total  float64
		wins   int
		loses  int
		volume float64
		sqn    float64
	)

	buffer := bytes.NewBuffer(nil)
	table := tablewriter.NewWriter(buffer)
	table.SetHeader([]string{"Pair", "Trades", "Win", "Loss", "% Win", "Payoff", "Pr Fact.", "SQN", "Profit", "Volume"})
	table.SetFooterAlignment(tablewriter.ALIGN_RIGHT)
	avgPayoff := 0.0
	avgProfitFactor := 0.0

	returns := make([]float64, 0)
	for _, summary := range n.orderController.Results {
		avgPayoff += summary.Payoff() * float64(len(summary.Win())+len(summary.Lose()))
		avgProfitFactor += summary.ProfitFactor() * float64(len(summary.Win())+len(summary.Lose()))
		table.Append([]string{
			summary.Pair,
			strconv.Itoa(len(summary.Win()) + len(summary.Lose())),
			strconv.Itoa(len(summary.Win())),
			strconv.Itoa(len(summary.Lose())),
			fmt.Sprintf("%.1f %%", float64(len(summary.Win()))/float64(len(summary.Win())+len(summary.Lose()))*100),
			fmt.Sprintf("%.3f", summary.Payoff()),
			fmt.Sprintf("%.3f", summary.ProfitFactor()),
			fmt.Sprintf("%.1f", summary.SQN()),
			fmt.Sprintf("%.2f", summary.Profit()),
			fmt.Sprintf("%.2f", summary.Volume),
		})
		total += summary.Profit()
		sqn += summary.SQN()
		wins += len(summary.Win())
		loses += len(summary.Lose())
		volume += summary.Volume

		returns = append(returns, summary.WinPercent()...)
		returns = append(returns, summary.LosePercent()...)
	}

	table.SetFooter([]string{
		"TOTAL",
		strconv.Itoa(wins + loses),
		strconv.Itoa(wins),
		strconv.Itoa(loses),
		fmt.Sprintf("%.1f %%", float64(wins)/float64(wins+loses)*100),
		fmt.Sprintf("%.3f", avgPayoff/float64(wins+loses)),
		fmt.Sprintf("%.3f", avgProfitFactor/float64(wins+loses)),
		fmt.Sprintf("%.1f", sqn/float64(len(n.orderController.Results))),
		fmt.Sprintf("%.2f", total),
		fmt.Sprintf("%.2f", volume),
	})
	table.Render()

	fmt.Println(buffer.String())
	fmt.Println("------ RETURN -------")
	totalReturn := 0.0
	returnsPercent := make([]float64, len(returns))
	for _, p := range returns {
		returnsPercent = append(returnsPercent, p*100)
		totalReturn += p
	}
	hist := histogram.Hist(15, returnsPercent)
	histogram.Fprint(os.Stdout, hist, histogram.Linear(10))
	fmt.Println()

	fmt.Println("------ CONFIDENCE INTERVAL (95%) -------")
	for pair, summary := range n.orderController.Results {
		fmt.Printf("| %s |\n", pair)
		returns := append(summary.WinPercent(), summary.LosePercent()...)
		returnsInterval := metrics.Bootstrap(returns, metrics.Mean, 10000, 0.95)
		payoffInterval := metrics.Bootstrap(returns, metrics.Payoff, 10000, 0.95)
		profitFactorInterval := metrics.Bootstrap(returns, metrics.ProfitFactor, 10000, 0.95)

		fmt.Printf("RETURN:      %.2f%% (%.2f%% ~ %.2f%%)\n",
			returnsInterval.Mean*100, returnsInterval.Lower*100, returnsInterval.Upper*100)
		fmt.Printf("PAYOFF:      %.2f (%.2f ~ %.2f)\n",
			payoffInterval.Mean, payoffInterval.Lower, payoffInterval.Upper)
		fmt.Printf("PROF.FACTOR: %.2f (%.2f ~ %.2f)\n",
			profitFactorInterval.Mean, profitFactorInterval.Lower, profitFactorInterval.Upper)
	}

	fmt.Println()

	if n.paperWallet != nil {
		n.paperWallet.Summary()
	}

}

// SaveReturns 将交易结果保存到CSV文件中
func (n NinjaBot) SaveReturns(outputDir string) error {
	// 遍历所有交易对的交易结果
	for _, summary := range n.orderController.Results {
		// 构建输出文件路径
		outputFile := fmt.Sprintf("%s/%s.csv", outputDir, summary.Pair)
		// 调用每个交易对的SaveReturns方法保存返回数据到CSV文件中
		if err := summary.SaveReturns(outputFile); err != nil {
			return err
		}
	}
	return nil
}

// onCandle 将蜡烛图数据推送到优先队列中
func (n *NinjaBot) onCandle(candle model.Candle) {
	n.priorityQueueCandle.Push(candle)
}

// processCandle 处理蜡烛图数据
func (n *NinjaBot) processCandle(candle model.Candle) {
	// 如果有纸钱包，将蜡烛图数据传递给纸钱包
	if n.paperWallet != nil {
		n.paperWallet.OnCandle(candle)
	}

	// 调用策略控制器的OnPartialCandle方法
	n.strategiesControllers[candle.Pair].OnPartialCandle(candle)
	// 如果蜡烛图完整，调用策略控制器的OnCandle方法和订单控制器的OnCandle方法
	if candle.Complete {
		n.strategiesControllers[candle.Pair].OnCandle(candle)
		n.orderController.OnCandle(candle)
	}
}

// processCandles 处理挂起的蜡烛图数据
// Process pending candles in buffer
func (n *NinjaBot) processCandles() {
	for item := range n.priorityQueueCandle.PopLock() {
		n.processCandle(item.(model.Candle))
	}
}

// backtestCandles 执行回测过程
// Start the backtest process and create a progress bar
// backtestCandles will process candles from a prirority queue in chronological order
func (n *NinjaBot) backtestCandles() {
	log.Info("[SETUP] Starting backtesting")

	// 创建进度条
	progressBar := progressbar.Default(int64(n.priorityQueueCandle.Len()))
	for n.priorityQueueCandle.Len() > 0 {
		item := n.priorityQueueCandle.Pop()

		candle := item.(model.Candle)
		// 如果有纸钱包，将蜡烛图数据传递给纸钱包
		if n.paperWallet != nil {
			n.paperWallet.OnCandle(candle)
		}

		// 调用策略控制器的OnPartialCandle方法和OnCandle方法
		n.strategiesControllers[candle.Pair].OnPartialCandle(candle)
		if candle.Complete {
			n.strategiesControllers[candle.Pair].OnCandle(candle)
		}

		// 更新进度条
		if err := progressBar.Add(1); err != nil {
			log.Warnf("update progressbar fail: %v", err)
		}
	}
}

// preload 预加载数据
// Before Ninjabot start, we need to load the necessary data to fill strategy indicators
// Then, we need to get the time frame and warmup period to fetch the necessary candles
func (n *NinjaBot) preload(ctx context.Context, pair string) error {
	if n.backtest {
		return nil
	}

	// 获取指定交易对的蜡烛图数据
	candles, err := n.exchange.CandlesByLimit(ctx, pair, n.strategy.Timeframe(), n.strategy.WarmupPeriod())
	if err != nil {
		return err
	}

	// 处理每个蜡烛图数据
	for _, candle := range candles {
		n.processCandle(candle)
	}

	// 预加载数据到数据源中
	n.dataFeed.Preload(pair, n.strategy.Timeframe(), candles)

	return nil
}

// Run 初始化机器人并启动
// Run will initialize the strategy controller, order controller, preload data and start the bot
func (n *NinjaBot) Run(ctx context.Context) error {
	for _, pair := range n.settings.Pairs {
		// 设置并订阅策略控制器
		// setup and subscribe strategy to data feed (candles)
		n.strategiesControllers[pair] = strategy.NewStrategyController(pair, n.strategy, n.orderController)

		// 预加载数据
		// preload candles for warmup period
		err := n.preload(ctx, pair)
		if err != nil {
			return err
		}

		// 订阅蜡烛图数据
		// link to ninja bot controller
		n.dataFeed.Subscribe(pair, n.strategy.Timeframe(), n.onCandle, false)

		// 启动策略控制器
		// start strategy controller
		n.strategiesControllers[pair].Start()
	}

	// 启动订单数据源和控制器
	// start order feed and controller
	n.orderFeed.Start()
	n.orderController.Start()
	defer n.orderController.Stop()
	if n.telegram != nil {
		n.telegram.Start()
	}

	// 启动数据源并接收新的蜡烛图数据
	// start data feed and receives new candles
	n.dataFeed.Start(n.backtest)

	// 根据环境启动处理蜡烛图数据的方法
	// start processing new candles for production or backtesting environment
	if n.backtest {
		n.backtestCandles()
	} else {
		n.processCandles()
	}

	return nil
}
