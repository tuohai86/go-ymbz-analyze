/**
 * 历史记录矩阵组件
 * 显示历史开奖和策略盈亏数据
 */

const HistoryMatrix = {
  name: 'HistoryMatrix',
  props: {
    logs: {
      type: Array,
      default: () => []
    },
    strategies: {
      type: Array,
      default: () => []
    }
  },
  data() {
    return {
      currentPage: 1,
      pageSize: 50,
      selectedLog: null,
      selectedStrategy: null,
      showModal: false
    };
  },
  computed: {
    // 策略表头
    strategyHeaders() {
      return this.strategies.map(s => Utils.cleanStrategyName(s.name));
    },
    
    // 总页数
    totalPages() {
      return Math.ceil(this.logs.length / this.pageSize) || 1;
    },
    
    // 当前页数据
    paginatedLogs() {
      const start = (this.currentPage - 1) * this.pageSize;
      const end = start + this.pageSize;
      return this.logs.slice(start, end);
    }
  },
  methods: {
    // 获取开奖结果颜色类
    getResultColorClass(result) {
      if (!result) return '';
      if (result.includes('红')) return 'text-danger';
      if (result.includes('绿')) return 'text-success';
      if (result.includes('黄')) return 'text-warning';
      if (result.includes('大三元') || result.includes('大四喜')) {
        return 'text-info';
      }
      return '';
    },
    
    // 获取盈亏数字
    getProfit(matrix, strategyName) {
      if (!matrix || !matrix[strategyName]) return '-';
      const item = matrix[strategyName];
      
      // 只显示实盘或状态变更的数据
      if (item.state === 1 || item.real_change !== 0) {
        return Utils.formatNumber(item.profit);
      }
      return '';
    },
    
    // 获取单元格类
    getCellClass(matrix, strategyName) {
      if (!matrix || !matrix[strategyName]) return '';
      const item = matrix[strategyName];
      
      // 只对实盘数据着色
      if (item.state === 1 || item.real_change !== 0) {
        if (item.profit > 0) return 'text-success font-semibold';
        if (item.profit < 0) return 'text-danger font-semibold';
      }
      return 'text-secondary';
    },
    
    // 显示详情
    showDetail(log, strategyName) {
      if (!log.matrix || !log.matrix[strategyName]) return;
      
      this.selectedLog = log;
      this.selectedStrategy = {
        name: strategyName,
        data: log.matrix[strategyName]
      };
      this.showModal = true;
    },
    
    // 关闭模态框
    closeModal() {
      this.showModal = false;
      setTimeout(() => {
        this.selectedLog = null;
        this.selectedStrategy = null;
      }, 300);
    },
    
    // 分页操作
    goToPage(page) {
      if (page >= 1 && page <= this.totalPages) {
        this.currentPage = page;
      }
    },
    
    // 导出CSV
    exportCSV() {
      const headers = ['期号', '开奖结果', ...this.strategyHeaders];
      const data = this.logs.map(log => {
        const row = {
          '期号': log.id,
          '开奖结果': log.res
        };
        this.strategyHeaders.forEach(name => {
          row[name] = this.getProfit(log.matrix, name);
        });
        return row;
      });
      
      const csv = Utils.generateCSV(data, headers);
      Utils.downloadFile(csv, `历史记录_${new Date().getTime()}.csv`, 'text/csv;charset=utf-8;');
    }
  },
  template: `
    <div class="glass-card">
      <div class="card-header">
        <h3 class="card-title">历史记录</h3>
        <button @click="exportCSV" class="btn btn-secondary btn-sm">
          导出CSV
        </button>
      </div>
      
      <div class="card-body p-0">
        <!-- 表格容器 -->
        <div style="overflow-x: auto; max-height: 600px;">
          <table style="width: 100%; border-collapse: collapse; font-size: 0.875rem;">
            <thead style="position: sticky; top: 0; background: var(--bg-secondary); z-index: 10;">
              <tr>
                <th style="padding: 0.75rem; text-align: left; border-bottom: 1px solid var(--border-color); min-width: 100px;">
                  期号
                </th>
                <th style="padding: 0.75rem; text-align: left; border-bottom: 1px solid var(--border-color); min-width: 120px;">
                  开奖结果
                </th>
                <th 
                  v-for="header in strategyHeaders"
                  :key="header"
                  style="padding: 0.75rem; text-align: center; border-bottom: 1px solid var(--border-color); min-width: 100px;"
                >
                  {{ header }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr 
                v-for="log in paginatedLogs"
                :key="log.id"
                style="border-bottom: 1px solid var(--border-color);"
                class="transition hover:bg-secondary"
              >
                <td style="padding: 0.75rem; font-family: var(--font-mono);">
                  {{ log.id }}
                </td>
                <td 
                  style="padding: 0.75rem; font-weight: 600;"
                  :class="getResultColorClass(log.res)"
                >
                  {{ log.res.split('[')[0] }}
                </td>
                <td 
                  v-for="strategyName in strategyHeaders"
                  :key="strategyName"
                  style="padding: 0.75rem; text-align: center; cursor: pointer; font-family: var(--font-mono);"
                  :class="getCellClass(log.matrix, strategyName)"
                  @click="showDetail(log, strategyName)"
                >
                  {{ getProfit(log.matrix, strategyName) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      
      <!-- 分页 -->
      <div class="card-footer">
        <div class="text-xs text-secondary">
          共 {{ logs.length }} 条记录
        </div>
        <div class="flex gap-sm items-center">
          <button 
            @click="goToPage(1)"
            :disabled="currentPage === 1"
            class="btn btn-secondary btn-sm"
          >
            首页
          </button>
          <button 
            @click="goToPage(currentPage - 1)"
            :disabled="currentPage === 1"
            class="btn btn-secondary btn-sm"
          >
            上一页
          </button>
          <span class="text-xs font-mono">
            {{ currentPage }} / {{ totalPages }}
          </span>
          <button 
            @click="goToPage(currentPage + 1)"
            :disabled="currentPage === totalPages"
            class="btn btn-secondary btn-sm"
          >
            下一页
          </button>
          <button 
            @click="goToPage(totalPages)"
            :disabled="currentPage === totalPages"
            class="btn btn-secondary btn-sm"
          >
            尾页
          </button>
        </div>
      </div>
      
      <!-- 详情模态框 -->
      <div 
        v-if="showModal"
        class="modal-backdrop"
        @click="closeModal"
      >
        <div class="modal zoom-in" @click.stop>
          <div class="card-header">
            <h3 class="card-title">{{ selectedStrategy?.name }}</h3>
            <span 
              :class="['badge', selectedStrategy?.data?.state === 1 ? 'badge-success' : 'badge-secondary']"
            >
              {{ selectedStrategy?.data?.state === 1 ? '实盘单' : '虚盘单' }}
            </span>
          </div>
          
          <div class="card-body">
            <div class="mb-md">
              <div class="text-sm text-secondary mb-xs">期号</div>
              <div class="font-mono font-semibold">{{ selectedLog?.id }}</div>
            </div>
            
            <div class="mb-md">
              <div class="text-sm text-secondary mb-xs">开奖结果</div>
              <div 
                class="font-semibold"
                :class="getResultColorClass(selectedLog?.res)"
              >
                {{ selectedLog?.res }}
              </div>
            </div>
            
            <div class="mb-md" v-if="selectedStrategy?.data?.pred">
              <div class="text-sm text-secondary mb-xs">推荐车型</div>
              <div class="flex flex-wrap gap-xs">
                <span
                  v-for="(car, index) in selectedStrategy.data.pred"
                  :key="index"
                  :class="['badge', Utils.getCarColorClass(car)]"
                >
                  {{ car }}
                </span>
              </div>
            </div>
            
            <div class="text-center">
              <div class="text-sm text-secondary mb-xs">盈亏</div>
              <div 
                class="text-3xl font-mono font-bold"
                :class="Utils.getValueColorClass(selectedStrategy?.data?.profit)"
              >
                {{ Utils.formatNumber(selectedStrategy?.data?.profit) }}
              </div>
            </div>
          </div>
          
          <div class="card-footer">
            <button @click="closeModal" class="btn btn-secondary" style="width: 100%;">
              关闭
            </button>
          </div>
        </div>
      </div>
    </div>
  `
};

// 导出组件
if (typeof module !== 'undefined' && module.exports) {
  module.exports = HistoryMatrix;
}
