/**
 * 策略状态卡片组件
 * 显示所有策略的实时状态
 */

const StrategyCards = {
  name: 'StrategyCards',
  props: {
    strategies: {
      type: Array,
      default: () => []
    }
  },
  methods: {
    // 获取状态文本
    getStateText(state) {
      return state === 1 ? '实盘中' : '观望';
    },
    
    // 获取状态类
    getStateClass(state) {
      return state === 1 ? 'real-pulse' : '';
    },
    
    // 获取盈利颜色类
    getProfitClass(profit) {
      return Utils.getValueColorClass(profit);
    },
    
    // 格式化盈利
    formatProfit(profit) {
      return Utils.formatNumber(profit);
    },
    
    // 清理策略名称
    cleanName(name) {
      return Utils.cleanStrategyName(name);
    },
    
    // 获取车型标签颜色类
    getCarTagClass(car) {
      return Utils.getCarColorClass(car);
    },
    
    // 简化车型名称
    simplifyCarName(car) {
      return Utils.simplifyCarName(car);
    }
  },
  template: `
    <div class="mb-lg">
      <h2 class="text-lg font-semibold mb-md">策略状态</h2>
      <div class="scroll-container">
        <div class="flex gap-md" style="min-width: min-content;">
          <div
            v-for="(strategy, index) in strategies"
            :key="index"
            class="glass-card hover-lift"
            :class="getStateClass(strategy.state)"
            style="min-width: 280px; max-width: 320px;"
          >
            <!-- 状态标签 -->
            <div class="flex justify-between items-start mb-md">
              <h3 class="text-base font-semibold m-0">
                {{ cleanName(strategy.name) }}
              </h3>
              <span 
                :class="['badge', strategy.state === 1 ? 'badge-success' : 'badge-secondary']"
              >
                {{ getStateText(strategy.state) }}
              </span>
            </div>
            
            <!-- 实盘盈利 -->
            <div class="mb-md">
              <div class="text-xs text-secondary mb-xs">实盘净利</div>
              <div 
                class="text-3xl font-mono font-bold"
                :class="getProfitClass(strategy.profit)"
              >
                {{ formatProfit(strategy.profit) }}
              </div>
            </div>
            
            <!-- 理论胜率 -->
            <div class="mb-md">
              <div class="text-xs text-secondary mb-xs">理论胜率</div>
              <div class="flex items-baseline gap-sm">
                <span class="text-xl font-semibold text-primary">
                  {{ strategy.rate }}%
                </span>
              </div>
            </div>
            
            <!-- 下期推荐 -->
            <div v-if="strategy.next && strategy.next.length > 0">
              <div class="text-xs text-secondary mb-xs">下期推荐</div>
              <div class="flex flex-wrap gap-xs">
                <span
                  v-for="(car, carIndex) in strategy.next"
                  :key="carIndex"
                  :class="['tag', getCarTagClass(car)]"
                >
                  {{ simplifyCarName(car) }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  `
};

// 导出组件
if (typeof module !== 'undefined' && module.exports) {
  module.exports = StrategyCards;
}
