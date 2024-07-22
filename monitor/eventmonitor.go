package monitor

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/elliotchance/pie/v2"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/go-multicall"

	"github.com/xiangxn/listener/config"
	"github.com/xiangxn/listener/database"
	"github.com/xiangxn/listener/dex"
	"github.com/xiangxn/listener/flashbots"
	si "github.com/xiangxn/listener/simulation"
	"github.com/xiangxn/listener/tools"
	dt "github.com/xiangxn/listener/types"
)

const POOL_BLACKLIST_FILE_NAME = "pool_blacklist.json"
const TOKEN_BLACKLIST_FILE_NAME = "token_blacklist.json"
const TOKEN_ERC20A_FILE_NAME = "token_erc20a.json"

var FieldTag = "monitor"

func createQuery(configEvents []config.DexConfig) ethereum.FilterQuery {
	length := len(configEvents)
	topics := make([][]common.Hash, 1)
	topics[0] = make([]common.Hash, length)

	var i int
	for i = 0; i < length; i++ {
		topics[0][i] = common.HexToHash(configEvents[i].Topic)
	}
	topics[0] = pie.Unique(topics[0])
	query := ethereum.FilterQuery{
		Topics: topics,
	}
	return query
}

// New 初始化eth 监控器
func New(opt *dt.Options) (dt.IMonitor, error) {
	ctx, cancel := context.WithCancel(context.Background())
	client, err := ethclient.DialContext(ctx, opt.Cfg.Rpcs.Ws)
	if err != nil {
		cancel()
		return nil, err
	}
	httpClient, err := ethclient.Dial(opt.Cfg.Rpcs.Http)
	if err != nil {
		cancel()
		return nil, err
	}
	m := &monitor{
		ctx:                ctx,
		cfg:                opt.Cfg,
		cancel:             cancel,
		cli:                client,
		httpClient:         httpClient,
		handler:            opt.Handler,
		logger:             opt.Logger,
		currentBlockNumber: 0,
		cacheEvents:        make(map[common.Address]types.Log),
		dexs:               make(map[string]IDex),
		baseBalance:        make(map[string]float64),
		gasPrice:           opt.Cfg.GasPrice, // default: 2GWei
		database: database.Actions{
			DB:     database.GetClient(opt.Cfg).Database(fmt.Sprintf("%slistener", opt.Cfg.NetName)),
			Mctx:   ctx,
			Logger: opt.Logger,
		},
		cipher: opt.Cipher,
	}
	// opt.Cipher = [32]byte{}
	// 获取chain id
	chainId, err := m.httpClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	m.chainId = chainId
	m.database.InitDataBase()
	m.factorys = m.GetListenFactory()
	m.multicall, err = multicall.Dial(ctx, opt.Cfg.Rpcs.Http)
	if err != nil {
		return nil, err
	}
	// 读取pool黑名单
	err = common.LoadJSON(POOL_BLACKLIST_FILE_NAME, &m.poolBlacklist)
	if err != nil {
		m.logger.Debugf("Failed to read file '%s'.", POOL_BLACKLIST_FILE_NAME)
	}
	m.poolBlacklist = pie.Unique(m.poolBlacklist)
	// 读取token黑名单
	err = common.LoadJSON(TOKEN_BLACKLIST_FILE_NAME, &m.tokenBlacklist)
	if err != nil {
		m.logger.Debugf("Failed to read file '%s'.", TOKEN_BLACKLIST_FILE_NAME)
	}
	m.tokenBlacklist = pie.Unique(m.tokenBlacklist)
	//读取erc20a的token列表(name字段是一个byte32)
	//
	err = common.LoadJSON(TOKEN_ERC20A_FILE_NAME, &m.tokenErc20a)
	if err != nil {
		m.logger.Debugf("Failed to read file '%s'.", TOKEN_ERC20A_FILE_NAME)
	}
	m.tokenErc20a = pie.Unique(m.tokenErc20a)
	// 添加新交易所时需要在这里添加对应的类型
	for _, d := range m.cfg.Dexs {
		switch d.Name {
		case "UniswapV2":
			m.dexs[d.Factory] = dex.GetDex[dex.UniswapV2](d, m)
		case "UniswapV3":
			m.dexs[d.Factory] = dex.GetDex[dex.UniswapV3](d, m)
		case "SushiSwap":
			m.dexs[d.Factory] = dex.GetDex[dex.SushiSwap](d, m)
		case "SushiSwapV3":
			m.dexs[d.Factory] = dex.GetDex[dex.UniswapV3](d, m)
		case "PancakeV3":
			m.dexs[d.Factory] = dex.GetDex[dex.PancakeV3](d, m)
		case "PancakeV2":
			m.dexs[d.Factory] = dex.GetDex[dex.PancakeV2](d, m)
		case "SolidlyV3":
			m.dexs[d.Factory] = dex.GetDex[dex.SolidlyV3](d, m)
		case "DefiSwap":
			m.dexs[d.Factory] = dex.GetDex[dex.DefiSwap](d, m)
		case "ShibaSwap":
			m.dexs[d.Factory] = dex.GetDex[dex.ShibaSwap](d, m)
		case "Thena":
			m.dexs[d.Factory] = dex.GetDex[dex.Thena](d, m)
		case "ApeSwap":
			m.dexs[d.Factory] = dex.GetDex[dex.ApeSwap](d, m)
		case "Biswap":
			m.dexs[d.Factory] = dex.GetDex[dex.Biswap](d, m)
		case "MDEX":
			m.dexs[d.Factory] = dex.GetDex[dex.MDEX](d, m)
		case "Aerodrome":
			m.dexs[d.Factory] = dex.GetDex[dex.Aerodrome](d, m)
		}
	}
	return m, nil
}

// 清理缓存的事件
func (m *monitor) clearCacheEvent() {
	for k := range m.cacheEvents {
		delete(m.cacheEvents, k)
	}
}

// 缓存Log,利用map去重处理数据:同一个池只处理最后一次事件
func (m *monitor) cacheEvent(vLog types.Log) {
	m.logger.WithField(FieldTag, "New Event").Debug(vLog.BlockNumber, vLog.Address, vLog.TxIndex, vLog.Index)
	if !vLog.Removed && (m.currentBlockNumber == 0 || m.currentBlockNumber == vLog.BlockNumber) {
		m.cacheEvents[vLog.Address] = vLog
	}
	m.currentBlockNumber = vLog.BlockNumber
}

// 预处理事件(包括拉取池信息与token信息)
func (m *monitor) preprocessEvent(logs []types.Log) []types.Log {
	//过滤掉池黑名单中的地址
	newLogs := pie.FilterNot(logs, func(value types.Log) bool {
		return pie.Contains(m.poolBlacklist, value.Address.Hex())
	})
	useful := dex.PreprocessEvent(m, m.factorys, newLogs)
	return useful
}

// 检查事件并开协程处理入库
func (m *monitor) checkEvent() {
	// fmt.Println("000")
	elength := len(m.cacheEvents)
	if elength == 0 {
		// m.logger.Info("Waiting for an event...")
		return
	}
	m.logger.Info("=============================开始处理事件===========================")
	events := make([]types.Log, 0, elength)
	for _, vLog := range m.cacheEvents {
		events = append(events, vLog)
	}
	// 清理缓存
	m.clearCacheEvent()
	m.currentBlockNumber = 0

	// 预处理事件
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		events = m.preprocessEvent(events)
	}()

	// 获取gas price
	wg.Add(1)
	go func() {
		defer wg.Done()
		gas, err := m.httpClient.SuggestGasPrice(m.ctx)
		if err == nil {
			m.gasPrice = tools.BigIntToFloat64(gas, 18)
		} else {
			m.logger.Error("获取gasPrice失败:", err)
		}
	}()
	t := time.Now()
	wg.Wait()
	m.logger.Info("本次预处理共", len(events), "个事件, 共用时: ", time.Since(t), fmt.Sprintf(", 最新GasPrice: %.18f", m.gasPrice))

	events = m.filterLog(events)
	m.logger.Info(fmt.Sprintf("有%d个事件需要计算套利。。。", len(events)))
	// 获取事件相关所有池的价格
	t = time.Now()
	m.fetchPrice(events)
	m.logger.Info(fmt.Sprintf("获取事件相关交易对价格, 共用时: %s", time.Since(t)))

	// 处理事件数据, 根据并发限制数来处理
	ch := make(chan struct{}, m.cfg.MaxConcurrent)
	for _, e := range events {
		ch <- struct{}{}
		go func(event types.Log) {
			m.processEvent(event)
			<-ch
		}(e)
	}
}

func (m *monitor) fetchPrice(events []types.Log) {
	addrs := pie.Map(events, func(event types.Log) string { return event.Address.Hex() })
	eventPools := m.database.GetSimplePools(addrs)
	var pools []dt.Pool
	for _, ep := range eventPools {
		ps := m.database.GetPoolsByTokens([]string{ep.Token0, ep.Token1})
		if len(ps) >= 2 {
			pools = append(pools, ps...)
		}
	}
	m.UpdatePrice(pools)
}

// 所有事件中，如果token0,token1一致，如eth/usdt,usdt/eth,只保留一个Log
func (m *monitor) filterLog(events []types.Log) (results []types.Log) {
	if len(events) < 1 {
		return
	}
	var addrs []string
	for _, event := range events {
		addrs = append(addrs, event.Address.Hex())
	}
	pools := m.database.GetSimplePools(addrs)
	pools = tools.Unique(pools)
	for _, pool := range pools {
		for _, event := range events {
			if pool.Address == event.Address.Hex() {
				results = append(results, event)
			}
		}
	}
	return
}

// 获取配置文件中的池工厂
func (m *monitor) GetListenFactory() []string {
	ls := make([]string, len(m.cfg.Dexs))
	for i, item := range m.cfg.Dexs {
		ls[i] = item.Factory
	}
	return ls
}

// 根据交易对地址获取交易所接口
func (m *monitor) GetDex(poolAddr string) (IDex, *dt.Pool) {
	pool := dex.GetFactory(m, poolAddr)
	if pool == nil {
		return nil, nil
	}
	return m.dexs[pool.Factory], pool
}

// 根据策略处理事件
func (m *monitor) processEvent(event types.Log) {
	idex, pool := m.GetDex(event.Address.Hex())
	if idex == nil || pool == nil {
		return
	}
	arbitrage, ok := m.handler.CalcArbitrage(m, event, pool, m.gasPrice*m.cfg.GasTimes)
	if ok {
		m.handler.Do(m, arbitrage)
	}
}

func (m *monitor) subscribeEvents(ctx context.Context) error {
	query := createQuery(m.cfg.Dexs)
	logs := make(chan types.Log)
	sub, err := m.cli.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return err
	}

	defer sub.Unsubscribe()

	timer := time.NewTimer(time.Hour) // 设置一个较长时间的初始计时器
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Subscription cancelled. Exiting subscription loop...")
			return nil
		case err := <-sub.Err():
			return err
		case vLog := <-logs:
			if !timer.Stop() && len(timer.C) > 0 {
				<-timer.C
			}
			m.cacheEvent(vLog)
			timer.Reset(time.Duration(m.cfg.EventWaitingTime) * time.Millisecond)
		case <-timer.C:
			timer.Reset(time.Hour)
			m.checkEvent()
		}
	}
}

func (m *monitor) ConfirmingTransaction() {
	for {
		txs := m.DB().GetTransactions(true, false)
		var wg sync.WaitGroup
		concurrent := make(chan struct{}, m.Config().MaxConcurrent)
		for _, tx := range txs {
			concurrent <- struct{}{}
			wg.Add(1)
			go func(mo *monitor, txr dt.Transaction) {
				defer wg.Done()
				receipt := si.GetReceipt(mo.httpClient, common.HexToHash(txr.Tx))
				if receipt != nil {
					income := m.caclIncome(m.httpClient, m.cfg.TraderContract, txr.Cost, txr.BaseToken)
					if receipt.Status == 1 {
						//实际运行时不再计算单笔收益。可以不定期查询余额与失败的消耗计算总收益
						mo.DB().UpdateTransaction(txr.Tx, true, receipt.GasUsed, receipt.EffectiveGasPrice.Uint64(), income, true)
					} else {
						mo.DB().UpdateTransaction(txr.Tx, true, receipt.GasUsed, receipt.EffectiveGasPrice.Uint64(), income, false)
						// m.checkFailTx(txr.BuyPool, txr.SellPool)
					}
				}
				<-concurrent
			}(m, tx)
		}
		time.Sleep(20 * time.Second) //20秒处理一次
	}
}

func (m *monitor) Run() {
	// Channel to listen for interrupt signal to gracefully shutdown the application
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(m.ctx)
	defer cancel()
	// 确认交易状态
	go m.ConfirmingTransaction()
	// Listen for stop signal in a separate goroutine
	go func() {
		<-stopChan
		cancel()
	}()

	for {
		err := m.subscribeEvents(ctx)
		if err != nil {
			m.logger.WithField(FieldTag, "subscribeEvents").Error(err)
		}
		select {
		case <-ctx.Done():
			m.logger.Info("Received stop signal. Exiting...")
			return
		default:
			// Continue to attempt reconnecting
			m.logger.WithField(FieldTag, "subscribeEvents").Info("Reconnecting after error...")
			time.Sleep(5 * time.Second) // Wait before attempting to reconnect
		}
	}
}

func (m *monitor) Cancel() {
	m.cli.Close()
	m.clearCacheEvent()
	m.cancel()
	database.Close()
}
func (m *monitor) GetPrivateKey() string {
	if m.privateKey != "" {
		return m.privateKey
	}
	m.privateKey = os.Getenv("PRIVATE_WIF")
	ciphertext, err := base32.StdEncoding.DecodeString(m.privateKey)
	if err != nil {
		panic(fmt.Sprintln("Error base32 decode:", err))
	}
	pk, err := tools.Decrypt(ciphertext, m.cipher[:])
	if err != nil {
		panic(fmt.Sprintln("Error decrypting:", err))
	}
	m.privateKey = string(pk)
	// m.cipher = [32]byte{}
	return m.privateKey
}
func (m *monitor) GetSignKey() string {
	if m.signKey != "" {
		return m.signKey
	}
	m.signKey = os.Getenv("SIGN_WIF")
	ciphertext, err := base32.StdEncoding.DecodeString(m.signKey)
	if err != nil {
		panic(fmt.Sprintln("Error base32 decode:", err))
	}
	pk, err := tools.Decrypt(ciphertext, m.cipher[:])
	if err != nil {
		panic(fmt.Sprintln("Error decrypting:", err))
	}
	m.signKey = string(pk)
	// m.cipher = [32]byte{}
	return m.signKey
}
func (m *monitor) GetHttpClient() *ethclient.Client {
	return m.httpClient
}
func (m *monitor) GetChainID() *big.Int {
	return m.chainId
}
func (m *monitor) GetContext() context.Context {
	return m.ctx
}
func (m *monitor) Logger() logrus.FieldLogger {
	return m.logger
}
func (m *monitor) DB() dt.IActions {
	return m.database
}
func (m *monitor) Multicall() *multicall.Caller {
	return m.multicall
}
func (m *monitor) Config() *config.Configuration {
	return &m.cfg
}

// 添加新的token黑名单,并保存到json文件
func (m *monitor) AddTokenBlacklist(addr string) {
	if !pie.Contains(m.tokenBlacklist, addr) {
		m.tokenBlacklist = append(m.tokenBlacklist, addr)
		tools.SaveJson(TOKEN_BLACKLIST_FILE_NAME, m.tokenBlacklist)
	}
}
func (m *monitor) GetTokenBlacklist() []string {
	return m.tokenBlacklist
}

func (m *monitor) AddPoolBlacklist(addr string) {
	if !pie.Contains(m.poolBlacklist, addr) {
		m.poolBlacklist = append(m.poolBlacklist, addr)
		tools.SaveJson(POOL_BLACKLIST_FILE_NAME, m.poolBlacklist)
	}
}
func (m *monitor) GetPoolBlacklist() []string {
	return m.poolBlacklist
}

// 添加旧ERC20合约地址
func (m *monitor) AddERC20A(addr common.Address) {
	if !pie.Contains(m.tokenErc20a, addr.Hex()) {
		m.tokenErc20a = append(m.tokenErc20a, addr.Hex())
		tools.SaveJson(TOKEN_ERC20A_FILE_NAME, m.tokenErc20a)
	}
}
func (m *monitor) GetERC20A() []string {
	return m.tokenErc20a
}

// 往TG发送消息
func (m *monitor) SendToTG(msg string) {
	telegramAPI := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", m.cfg.TG.Token)
	body := dt.TelegramRequestBody{
		ChatID: m.cfg.TG.ChatID,
		Text:   fmt.Sprintf("%s %s", m.cfg.NetName, msg),
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		m.logger.Error("SendToTG Error: ", err)
		return
	}

	req, err := http.NewRequest("POST", telegramAPI, bytes.NewBuffer(bodyBytes))
	if err != nil {
		m.logger.Error("SendToTG Error: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Error("SendToTG Error: ", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("SendToTG Error: ", fmt.Errorf("failed to send message: %s", resp.Status))
		return
	}
}

func (m *monitor) UpdatePrice(pools []dt.Pool) {
	mcContract, err := multicall.NewContract(dex.BlockNumberABI, multicall.DefaultAddress)
	if err != nil {
		m.logger.Error("UpdatePrice 0:", err)
		return
	}
	var startIndex = 2
	var calls []*multicall.Call
	calls = append(calls, mcContract.NewCall(new(dt.ResBigInt), "getBlockNumber").AllowFailure())
	calls = append(calls, mcContract.NewCall(new(dt.ResBigInt), "getBasefee").AllowFailure())
	if m.cfg.TraderContract != "" {
		cbs := m.handler.CreateBalanceCalls(dex.TokenABI, common.HexToAddress(m.cfg.TraderContract))
		length := len(cbs)
		if length > 0 {
			calls = append(calls, cbs...)
		}
		startIndex += length
	}
	for _, p := range pools {
		idex := m.dexs[p.Factory]
		if idex == nil {
			continue
		}
		call := idex.CreatePriceCall(&p)
		if len(call) > 0 {
			calls = append(calls, call...)
		}
	}

	t := time.Now()
	results, err := m.multicall.Call(nil, calls...)
	m.logger.Info(fmt.Sprintf("UpdatePrice Multicall调用, 共%d个Call, 共用时%s", len(calls), time.Since(t)))
	if err != nil {
		m.logger.Error("UpdatePrice 1:", err)
		return
	}
	blockNumber := calls[0].Outputs.(*dt.ResBigInt).Int
	m.baseFee = calls[1].Outputs.(*dt.ResBigInt).Int
	if startIndex > 2 {
		bts, decs := m.handler.GetBaseTokens()
		for i, bt := range bts {
			balance := calls[i+1].Outputs.(*dt.ResBigInt).Int
			m.baseBalance[bt] = tools.BigIntToFloat64(balance, decs[i])
		}
	}

	var wg sync.WaitGroup
	taskChan := make(chan dt.Pair)
	resCalls := results[startIndex:]
	for i := 0; i < len(resCalls); {
		c := resCalls[i]
		pIndex := pie.FindFirstUsing(pools, func(v dt.Pool) bool { return v.Address == c.CallName })
		p := pools[pIndex]
		d := m.dexs[p.Factory]
		pcs := resCalls[i : i+d.PriceCallCount()]
		wg.Add(1)
		go func(res []*multicall.Call, p *dt.Pool, dd IDex, bn *big.Int) {
			pair := dd.CalcPrice(res, bn, p)
			taskChan <- pair
			wg.Done()
		}(pcs, &p, d, blockNumber)
		i += d.PriceCallCount()
	}
	go func() {
		wg.Wait()
		close(taskChan)
	}()
	var pairs []dt.Pair
	for pair := range taskChan {
		pairs = append(pairs, pair)
	}
	m.database.SavePairs(pairs)
}

func (m *monitor) GetUseGas(buyPool, sellPool *dt.Pair, amount float64) int64 {
	minGas, maxGas := m.database.GetGas(buyPool.Pool, sellPool.Pool)
	if minGas == 0 || maxGas == 0 {
		return 300000 //如果数据库中不存在数据，就以最高gas消耗来计算
	}
	if amount <= m.baseBalance[sellPool.Token0] {
		return minGas
	} else {
		return maxGas
	}
}

func (m *monitor) Swap(client *ethclient.Client, params dt.SwapParams, traderContract string, simulation bool, cost float64, baseDec uint64) (txHash common.Hash) {
	if traderContract == "" { //如果不配置套利合约就不执行调用
		m.logger.Info("No arbitrage contract is configured.")
		return
	}

	// fmt.Println("params.amount:", params.Amount)
	// fmt.Println("params.position:", params.Position)
	// fmt.Println("params.deadline:", params.Deadline)
	// fmt.Println("params.buyPool:", params.BuyPool)
	// fmt.Println("params.sellPool:", params.SellPool)
	// fmt.Println("params.borrowPool:", params.Borrow)
	// fmt.Println("params.baseToken:", params.BaseToken)

	if params.BuyType == 0 {
		idex, _ := m.GetDex(params.BuyPool)
		params.BuyType = idex.GetType()
	}
	if params.SellType == 0 {
		idex, _ := m.GetDex(params.SellPool)
		params.SellType = idex.GetType()
	}

	privateKey := si.GetPrivateKey(m.GetPrivateKey())
	fromAddress := si.GetAddress(privateKey)

	ctx, cancel := context.WithCancel(m.ctx)
	defer cancel()

	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		m.Logger().WithField(FieldTag, "Swap1").Error(err)
		return
	}
	gasPrice := tools.Float64ToBigInt(params.GasPrice, 18)

	hash := crypto.Keccak256Hash([]byte("swap()")).Hex()
	methodID := hash[:10]
	amount := tools.Float64ToBigInt(params.Amount, baseDec)

	tmp := new(big.Int).Lsh(big.NewInt(int64(params.Deadline)), 72)
	tmp = tmp.Or(tmp, new(big.Int).Lsh(big.NewInt(int64(params.Position)), 64))
	tmp = tmp.Or(tmp, new(big.Int).Lsh(big.NewInt(int64(params.BuyType)), 48))
	tmp = tmp.Or(tmp, new(big.Int).Lsh(big.NewInt(int64(params.SellType)), 32))
	tmp = tmp.Or(tmp, new(big.Int).Lsh(big.NewInt(int64(params.BuyFee)), 16))
	tmp = tmp.Or(tmp, big.NewInt(int64(params.SellFee)))

	var data []byte
	data = append(data, hexutil.MustDecode(methodID)...)
	data = append(data, common.LeftPadBytes(common.HexToAddress(params.BuyPool).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.HexToAddress(params.SellPool).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.HexToAddress(params.BaseToken).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.HexToAddress(params.Borrow).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(tmp.Bytes(), 32)...)

	// fmt.Printf("data: %x", data)
	var tx *types.Transaction
	var signedTx *types.Transaction
	// var err error
	if m.cfg.EIP1559 {
		to := common.HexToAddress(traderContract)
		maxPriorityFeePerGas := new(big.Int).Sub(gasPrice, m.baseFee)
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:   m.chainId,
			Nonce:     nonce,
			GasFeeCap: gasPrice,
			GasTipCap: maxPriorityFeePerGas,
			Gas:       m.cfg.GasLimit * uint64(m.cfg.GasTimes),
			To:        &to,
			Value:     big.NewInt(0),
			Data:      data,
		})
		signedTx, err = types.SignTx(tx, types.NewLondonSigner(m.chainId), privateKey)
	} else {
		tx = types.NewTransaction(nonce, common.HexToAddress(traderContract), big.NewInt(0), m.cfg.GasLimit*uint64(m.cfg.GasTimes), gasPrice, data)
		signedTx, err = types.SignTx(tx, types.NewEIP155Signer(m.chainId), privateKey)
	}

	if err != nil {
		m.Logger().WithField(FieldTag, "Swap3").Error(err)
		return
	}

	ok := true
	confirm := false
	if m.cfg.Simulation.Enable {
		err = client.SendTransaction(ctx, signedTx)
	} else {
		switch m.cfg.Rpcs.Flashbots {
		case "alchemy":
			err = m.sendPrivateTransaction(ctx, signedTx, params.Deadline, m.cfg.Rpcs.Http)
		case "flashbot":
			err = m.sendPrivateTransaction(ctx, signedTx, params.Deadline, "")
		default:
			err = client.SendTransaction(ctx, signedTx)
		}
	}

	if err != nil {
		if strings.Contains(err.Error(), "insufficient funds for gas *") {
			go m.SendToTG(fmt.Sprintf("机器人余额不足: \nhttps://etherscan.io/address/%s", fromAddress))
		} else if !pie.Contains([]string{"D", "E"}, err.Error()) {
			m.checkFailTx(params.BuyPool, params.SellPool)
		}
		m.Logger().WithField(FieldTag, "Swap4").Error(err)
		ok = false
		confirm = true
	}

	txHash = signedTx.Hash()
	m.database.SaveTransaction(dt.Transaction{
		Tx:         txHash.Hex(),
		Ok:         ok,
		Confirm:    confirm,
		Cost:       cost,
		Simulation: simulation,
		BuyPool:    params.BuyPool,
		SellPool:   params.SellPool,
		BaseToken:  params.BaseToken,
		CreatedAt:  time.Now(),
	})
	return
}

func (m *monitor) sendPrivateTransaction(ctx context.Context, signedTx *types.Transaction, maxBlock uint64, url string) error {
	data, err := signedTx.MarshalBinary()
	if err != nil {
		return err
	}
	param := flashbots.ParamsPrivateTransaction{
		Tx:             hexutil.Encode(data),
		MaxBlockNumber: fmt.Sprintf("0x%x", maxBlock),
	}
	param.Preferences.Fast = true
	// param.Preferences.Privacy.Builders = []string{"flashbots", "beaverbuild.org", "f1b.io", "rsync", "builder0x69", "Titan", "EigenPhi", "BTCS", "JetBuilder"}

	privateKey := si.GetPrivateKey(m.GetSignKey())
	fromAddress := si.GetAddress(privateKey)
	resp, err := flashbots.FlashbotRequest(ctx, privateKey, &fromAddress, url, "eth_sendPrivateTransaction", param)
	if err != nil {
		return errors.Wrap(err, "flashbot private TX request")
	}

	rr := &flashbots.SendPrivateTransactionResponse{}

	err = json.Unmarshal(resp, rr)
	if err != nil {
		return errors.Wrapf(err, "unmarshal flashbot response:%v", string(resp))
	}
	if rr.Error.Code != 0 {
		errStr := fmt.Sprintf("flashbot request returned an error:%+v,%v block:%v", rr.Error, rr.Message, maxBlock)
		return errors.New(errStr)
	}

	return nil
}

// 在数据库中检查失败的交易，如果失败次数>=1就把池加入黑名单
func (m *monitor) checkFailTx(buyPool, sellPool string) {
	failCount := m.database.GetFailTransacttionCount(buyPool, sellPool)
	if failCount >= 1 {
		pool := m.database.GetSimplePool(buyPool)
		baseToken := m.handler.GetBaseToken(pool.Token0, pool.Token1)
		if baseToken == pool.Token0 {
			m.AddTokenBlacklist(pool.Token1)
		} else {
			m.AddTokenBlacklist(pool.Token0)
		}
	}
}

func (m *monitor) DoSwap(params dt.SwapParams) {
	if m.cfg.Simulation.Enable { //模拟交易
		ctx, cancel := context.WithCancel(m.ctx)
		richAddress := m.cfg.Simulation.Funds
		privateKey := si.GetPrivateKey(m.GetPrivateKey())
		testAddress := si.GetAddress(privateKey)
		rpcURL := m.cfg.Rpcs.Http
		port := si.RandomPort()
		block := params.Deadline
		si.StartAnvil(ctx, rpcURL, block-1, port)
		defer cancel()
		si.WaitForAnvil(port)

		si.Impersonate(port, richAddress)
		//给测试号一点eth
		si.ImpersonateTransferETH(port, richAddress, testAddress.Hex(), 1)
		// 在分叉上部署套利合约
		traderContract := si.DeployTrader(port, privateKey)
		//给合约一点basetoken
		cost := 0.1
		dec := m.handler.GetBaseDecimals(params.BaseToken)
		si.ImpersonateTransfer(port, params.BaseToken, richAddress, traderContract, cost, dec)
		si.StopImpersonate(port, richAddress)
		//为params的Deadline添加三个块，因为anvil为自动开采区块(每调用一次send就会开采一个块)
		params.Deadline += 3
		//开始模拟交易
		txHash := m.Swap(si.GetClient(port), params, traderContract, true, cost, dec)
		// 计算收益情况
		for {
			receipt := si.GetReceipt(si.GetClient(port), txHash)
			if receipt != nil {
				income := m.caclIncome(si.GetClient(port), traderContract, cost, params.BaseToken)
				if receipt.Status == 1 {
					m.database.UpdateTransaction(txHash.Hex(), true, receipt.GasUsed, receipt.EffectiveGasPrice.Uint64(), income, true)
				} else {
					m.database.UpdateTransaction(txHash.Hex(), true, receipt.GasUsed, receipt.EffectiveGasPrice.Uint64(), income, false)
					// m.checkFailTx(params.BuyPool, params.SellPool)
				}
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	} else { //真实交易
		// 真实交易时不再记录每笔成本
		dec := m.handler.GetBaseDecimals(params.BaseToken)
		cost := m.baseBalance[params.BaseToken]
		m.Swap(m.httpClient, params, m.cfg.TraderContract, false, cost, dec)
	}
}

// 计算income或者转换余额
func (m *monitor) caclIncome(client *ethclient.Client, traderContract string, cost float64, bt string) (income float64) {
	baseToken := m.database.GetToken(bt)
	bBig := si.BalanceOf(client, baseToken.Address, traderContract)
	balance := tools.BigIntToFloat64(bBig, baseToken.Decimals)
	income = balance - cost
	return
}

func (m *monitor) TestEvent(event types.Log) {
	m.processEvent(event)
}
