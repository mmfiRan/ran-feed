package oss

// StrategyFactory 策略工厂
type StrategyFactory struct {
	strategies map[string]Strategy
}

func NewStrategyFactory() *StrategyFactory {
	return &StrategyFactory{strategies: make(map[string]Strategy)}
}

func (f *StrategyFactory) Register(name string, strategy Strategy) {
	f.strategies[name] = strategy
}

func (f *StrategyFactory) GetStrategy(name string) (Strategy, bool) {
	strategy, ok := f.strategies[name]
	return strategy, ok
}

func (f *StrategyFactory) MustGetStrategy(name string) Strategy {
	strategy, ok := f.strategies[name]
	if !ok {
		panic("不支持的OSS提供者: " + name)
	}
	return strategy
}
