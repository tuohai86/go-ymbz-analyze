/**
 * 预测推荐面板组件
 * 显示下期实盘策略推荐
 */

const PredictionPanel = {
  name: 'PredictionPanel',
  props: {
    strategies: {
      type: Array,
      default: () => []
    },
    predictions: {
      type: Object,
      default: () => ({})
    }
  },
  data() {
    return {
      copied: false
    };
  },
  computed: {
    // 获取实盘策略
    activeStrategies() {
      return this.strategies.filter(s => s.state === 1);
    },
    
    // 合并所有实盘策略的推荐
    allPredictions() {
      const predSet = new Set();
      this.activeStrategies.forEach(strategy => {
        if (strategy.next && Array.isArray(strategy.next)) {
          strategy.next.forEach(car => predSet.add(car));
        }
      });
      return Array.from(predSet);
    },
    
    // 计算总下注金额
    totalAmount() {
      return Object.keys(this.predictions).length * 100;
    }
  },
  methods: {
    // 获取车型标签颜色类
    getCarTagClass(car) {
      return Utils.getCarColorClass(car);
    },
    
    // 简化车型名称
    simplifyCarName(car) {
      return Utils.simplifyCarName(car);
    },
    
    // 复制推荐到剪贴板
    async copyPredictions() {
      const text = this.allPredictions.join(', ');
      const success = await Utils.copyToClipboard(text);
      
      if (success) {
        this.copied = true;
        setTimeout(() => {
          this.copied = false;
        }, 2000);
      }
    }
  },
  template: `
    <div class="glass-card h-full">
      <div class="card-header">
        <h3 class="card-title">下期推荐</h3>
        <span class="badge badge-primary">实盘策略</span>
      </div>
      
      <div class="card-body">
        <!-- 实盘策略数量 -->
        <div class="mb-md">
          <div class="text-xs text-secondary mb-xs">实盘策略数量</div>
          <div class="text-2xl font-bold text-primary">
            {{ activeStrategies.length }} 个
          </div>
        </div>
        
        <!-- 推荐车型 -->
        <div class="mb-md" v-if="allPredictions.length > 0">
          <div class="text-xs text-secondary mb-xs">推荐车型</div>
          <div class="flex flex-wrap gap-sm">
            <span
              v-for="(car, index) in allPredictions"
              :key="index"
              :class="['tag', getCarTagClass(car), 'hover-scale']"
            >
              {{ simplifyCarName(car) }}
            </span>
          </div>
        </div>
        
        <!-- 下注金额 -->
        <div class="mb-md" v-if="Object.keys(predictions).length > 0">
          <div class="text-xs text-secondary mb-xs">建议总投入</div>
          <div class="text-xl font-mono font-semibold text-warning">
            {{ totalAmount }} 元
          </div>
        </div>
        
        <!-- 空状态 -->
        <div v-if="activeStrategies.length === 0" class="text-center py-lg">
          <div class="text-secondary text-sm">
            暂无实盘策略
          </div>
        </div>
      </div>
      
      <!-- 操作按钮 -->
      <div class="card-footer" v-if="allPredictions.length > 0">
        <button 
          @click="copyPredictions"
          :class="['btn', copied ? 'btn-success' : 'btn-primary', 'btn-sm']"
          style="width: 100%;"
        >
          {{ copied ? '✓ 已复制' : '复制推荐' }}
        </button>
      </div>
    </div>
  `
};

// 导出组件
if (typeof module !== 'undefined' && module.exports) {
  module.exports = PredictionPanel;
}
