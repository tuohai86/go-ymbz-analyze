/**
 * Vue应用主逻辑
 * 整合所有组件，管理全局状态
 */

const { createApp } = Vue;

createApp({
  components: {
    'countdown-hero': CountdownHero,
    'strategy-cards': StrategyCards,
    'prediction-panel': PredictionPanel,
    'history-matrix': HistoryMatrix
  },
  
  data() {
    return {
      // 连接状态
      connected: false,
      
      // 实时数据
      lid: '',
      nextLid: '',
      lastRes: '',
      countdown: 34,
      leaderboard: [],
      logs: [],
      predictions: {},
      
      // 加载状态
      loading: true,
      error: null,
      
      // 轮询停止函数
      stopPolling: null
    };
  },
  
  computed: {
    // 是否有数据
    hasData() {
      return this.leaderboard.length > 0;
    }
  },
  
  methods: {
    /**
     * 处理状态更新
     */
    handleStatusUpdate(data) {
      this.connected = true;
      this.lid = data.lid || '';
      this.nextLid = data.next_lid || (this.lid ? String(Number(this.lid) + 1) : '');
      this.lastRes = data.last_res || '';
      this.countdown = data.countdown !== undefined ? data.countdown : 34;
      this.leaderboard = data.leaderboard || [];
      this.logs = data.logs || [];
      this.loading = false;
      this.error = null;
    },
    
    /**
     * 加载预测数据
     */
    async loadPredictions() {
      const result = await API.getPredictions();
      if (result.success) {
        this.predictions = result.data.predictions || {};
      }
    },
    
    /**
     * 初始化数据
     */
    async initData() {
      this.loading = true;
      
      // 获取初始状态
      const result = await API.getStatus();
      if (result.success) {
        this.handleStatusUpdate(result.data);
      } else {
        this.error = result.error;
        this.loading = false;
        this.connected = false;
      }
      
      // 加载预测
      await this.loadPredictions();
    },
    
    /**
     * 启动轮询
     */
    startDataPolling() {
      this.stopPolling = API.startPolling((data) => {
        this.handleStatusUpdate(data);
        
        // 每次更新时也刷新预测
        this.loadPredictions();
      }, 2000);
    },
    
    /**
     * 格式化错误信息
     */
    formatError(error) {
      if (typeof error === 'string') return error;
      return error?.message || '未知错误';
    }
  },
  
  mounted() {
    console.log('🚀 奔驰宝马分析系统启动');
    
    // 初始化数据
    this.initData();
    
    // 启动轮询
    this.startDataPolling();
    
    // 监听页面可见性，优化性能
    document.addEventListener('visibilitychange', () => {
      if (document.hidden) {
        // 页面不可见时停止轮询
        if (this.stopPolling) {
          this.stopPolling();
          this.stopPolling = null;
        }
      } else {
        // 页面可见时重新启动轮询
        if (!this.stopPolling) {
          this.initData();
          this.startDataPolling();
        }
      }
    });
  },
  
  beforeUnmount() {
    // 清理轮询
    if (this.stopPolling) {
      this.stopPolling();
    }
  },
  
  template: `
    <div id="app-container" class="container" style="padding-top: 2rem; padding-bottom: 2rem;">
      <!-- 加载状态 -->
      <div v-if="loading" class="text-center" style="padding: 4rem 0;">
        <div class="loading-spinner" style="margin: 0 auto 1rem;"></div>
        <div class="text-secondary">加载中...</div>
      </div>
      
      <!-- 错误状态 -->
      <div v-else-if="error" class="glass-card text-center" style="padding: 2rem;">
        <div class="text-danger text-lg mb-md">连接失败</div>
        <div class="text-secondary text-sm mb-lg">{{ formatError(error) }}</div>
        <button @click="initData" class="btn btn-primary">重试</button>
      </div>
      
      <!-- 主内容 -->
      <div v-else>
        <!-- Hero区域 -->
        <countdown-hero
          :next-lid="nextLid"
          :countdown="countdown"
          :last-res="lastRes"
          :connected="connected"
        ></countdown-hero>
        
        <!-- 策略卡片 -->
        <strategy-cards
          :strategies="leaderboard"
        ></strategy-cards>
        
        <!-- 数据可视化和预测区域 -->
        <div class="grid grid-cols-3 gap-lg mb-lg">
          <!-- 数据占位（2列） -->
          <div class="glass-card" style="grid-column: span 2;">
            <div class="card-header">
              <h3 class="card-title">数据概览</h3>
            </div>
            <div class="card-body">
              <!-- 统计卡片 -->
              <div class="grid grid-cols-3 gap-md">
                <div class="text-center">
                  <div class="text-xs text-secondary mb-xs">总期数</div>
                  <div class="text-2xl font-bold text-primary">
                    {{ logs.length }}
                  </div>
                </div>
                <div class="text-center">
                  <div class="text-xs text-secondary mb-xs">实盘策略</div>
                  <div class="text-2xl font-bold text-success">
                    {{ leaderboard.filter(s => s.state === 1).length }}
                  </div>
                </div>
                <div class="text-center">
                  <div class="text-xs text-secondary mb-xs">观望策略</div>
                  <div class="text-2xl font-bold text-secondary">
                    {{ leaderboard.filter(s => s.state === 0).length }}
                  </div>
                </div>
              </div>
              
              <!-- 策略盈利排行 -->
              <div class="mt-lg">
                <h4 class="text-sm text-secondary mb-md">实盘盈利排行</h4>
                <div class="flex flex-col gap-sm">
                  <div 
                    v-for="(strategy, index) in leaderboard.slice().sort((a, b) => b.profit - a.profit).slice(0, 5)"
                    :key="index"
                    class="flex justify-between items-center p-sm rounded"
                    style="background: rgba(255, 255, 255, 0.03);"
                  >
                    <div class="flex items-center gap-sm">
                      <span class="text-xs text-secondary" style="width: 20px;">{{ index + 1 }}</span>
                      <span class="text-sm">{{ Utils.cleanStrategyName(strategy.name) }}</span>
                    </div>
                    <span 
                      class="font-mono font-semibold"
                      :class="Utils.getValueColorClass(strategy.profit)"
                    >
                      {{ Utils.formatNumber(strategy.profit) }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 预测面板（1列） -->
          <prediction-panel
            :strategies="leaderboard"
            :predictions="predictions"
          ></prediction-panel>
        </div>
        
        <!-- 历史记录矩阵 -->
        <history-matrix
          :logs="logs"
          :strategies="leaderboard"
        ></history-matrix>
      </div>
    </div>
  `
}).mount('#app');
