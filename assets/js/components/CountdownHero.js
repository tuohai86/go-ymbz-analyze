/**
 * 倒计时Hero组件
 * 显示下期期号、倒计时、上期结果
 */

const CountdownHero = {
  name: 'CountdownHero',
  props: {
    nextLid: {
      type: String,
      default: '--'
    },
    countdown: {
      type: Number,
      default: 34
    },
    lastRes: {
      type: String,
      default: ''
    },
    connected: {
      type: Boolean,
      default: false
    }
  },
  computed: {
    // 计算进度百分比
    progress() {
      return Utils.calculateProgress(this.countdown, 34);
    },

    // 倒计时显示文本
    countdownText() {
      return `${this.countdown}s`;
    },

    // 环形进度条参数
    circularParams() {
      const radius = 60;
      const strokeWidth = 8;
      return Utils.getCircularProgress(radius, strokeWidth);
    },

    // 计算stroke-dashoffset
    strokeDashoffset() {
      return this.circularParams.getDashoffset(this.progress);
    },

    // 上期结果颜色类
    lastResColorClass() {
      if (!this.lastRes) return '';
      if (this.lastRes.includes('红')) return 'text-danger';
      if (this.lastRes.includes('绿')) return 'text-success';
      if (this.lastRes.includes('黄')) return 'text-warning';
      if (this.lastRes.includes('大三元') || this.lastRes.includes('大四喜')) {
        return 'text-info';
      }
      return '';
    }
  },
  template: `
    <div class="glass-card p-xl mb-lg fade-in-down">
      <!-- 顶部状态栏 -->
      <div class="flex justify-between items-center mb-md">
        <div class="flex items-center gap-sm">
          <h1 class="text-sm text-secondary m-0">奔驰宝马分析系统</h1>
        </div>
        <div class="status-indicator">
          <span :class="['status-dot', connected ? 'success' : 'danger']"></span>
          <span class="text-xs" :class="connected ? 'text-success' : 'text-danger'">
            {{ connected ? '实时连接' : '连接断开' }}
          </span>
        </div>
      </div>
      
      <!-- 主内容区域 -->
      <div class="grid grid-cols-2 gap-lg items-center">
        <!-- 左侧：期号和进度条 -->
        <div>
          <div class="text-sm text-secondary mb-sm">下期期号</div>
          <div class="flex items-baseline gap-md mb-lg">
            <h1 class="text-4xl font-mono font-bold text-primary m-0">
              {{ nextLid }}
            </h1>
            <span class="badge badge-primary">
              {{ countdownText }}
            </span>
          </div>
          
          <!-- 线性进度条 -->
          <div class="progress-bar mb-md">
            <div 
              class="progress-bar-fill" 
              :style="{ width: progress + '%' }"
            ></div>
          </div>
          
          <!-- 上期结果 -->
          <div class="text-xs text-secondary">
            上期结果: 
            <span class="font-semibold" :class="lastResColorClass">
              {{ lastRes || '--' }}
            </span>
          </div>
        </div>
        
        <!-- 右侧：环形倒计时 -->
        <div class="flex justify-center">
          <div class="circular-progress">
            <svg :width="120" :height="120">
              <circle
                class="circular-progress-bg"
                :cx="60"
                :cy="60"
                :r="circularParams.normalizedRadius"
              ></circle>
              <circle
                class="circular-progress-bar"
                :cx="60"
                :cy="60"
                :r="circularParams.normalizedRadius"
                :stroke-dasharray="circularParams.circumference"
                :stroke-dashoffset="strokeDashoffset"
              ></circle>
            </svg>
            <div class="circular-progress-text">
              {{ countdown }}
            </div>
          </div>
        </div>
      </div>
    </div>
  `
};

// 导出组件
if (typeof module !== 'undefined' && module.exports) {
  module.exports = CountdownHero;
}
