/**
 * 工具函数模块
 * 包含格式化、颜色判断、时间处理等通用函数
 */

const Utils = {
  /**
   * 格式化数字（添加正负号）
   * @param {number} num - 数字
   * @returns {string} 格式化后的字符串
   */
  formatNumber(num) {
    if (num === null || num === undefined || num === 0) return '0';
    return num > 0 ? `+${num}` : `${num}`;
  },
  
  /**
   * 格式化百分比
   * @param {number} value - 数值
   * @param {number} total - 总数
   * @param {number} decimals - 小数位数
   * @returns {string} 百分比字符串
   */
  formatPercent(value, total, decimals = 1) {
    if (!total || total === 0) return '0%';
    const percent = (value / total * 100).toFixed(decimals);
    return `${percent}%`;
  },
  
  /**
   * 根据车型名称获取颜色类
   * @param {string} carName - 车型名称
   * @returns {string} CSS类名
   */
  getCarColorClass(carName) {
    if (!carName) return '';
    if (carName.includes('红')) return 'tag-red';
    if (carName.includes('绿')) return 'tag-green';
    if (carName.includes('黄')) return 'tag-yellow';
    return '';
  },
  
  /**
   * 根据数值获取颜色类（正负）
   * @param {number} value - 数值
   * @returns {string} CSS类名
   */
  getValueColorClass(value) {
    if (value > 0) return 'text-success';
    if (value < 0) return 'text-danger';
    return 'text-secondary';
  },
  
  /**
   * 根据数值获取徽章类型
   * @param {number} value - 数值
   * @returns {string} CSS类名
   */
  getBadgeClass(value) {
    if (value > 0) return 'badge-success';
    if (value < 0) return 'badge-danger';
    return 'badge-secondary';
  },
  
  /**
   * 格式化时间戳为HH:MM:SS
   * @param {number} seconds - 秒数
   * @returns {string} 格式化的时间
   */
  formatTime(seconds) {
    if (seconds < 0) seconds = 0;
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = Math.floor(seconds % 60);
    
    if (h > 0) {
      return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
    }
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  },
  
  /**
   * 计算倒计时百分比
   * @param {number} current - 当前秒数
   * @param {number} total - 总秒数
   * @returns {number} 百分比（0-100）
   */
  calculateProgress(current, total) {
    if (total === 0) return 0;
    return Math.max(0, Math.min(100, (current / total) * 100));
  },
  
  /**
   * 防抖函数
   * @param {Function} func - 要防抖的函数
   * @param {number} wait - 等待时间（毫秒）
   * @returns {Function} 防抖后的函数
   */
  debounce(func, wait = 300) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  },
  
  /**
   * 节流函数
   * @param {Function} func - 要节流的函数
   * @param {number} limit - 时间限制（毫秒）
   * @returns {Function} 节流后的函数
   */
  throttle(func, limit = 300) {
    let inThrottle;
    return function executedFunction(...args) {
      if (!inThrottle) {
        func(...args);
        inThrottle = true;
        setTimeout(() => (inThrottle = false), limit);
      }
    };
  },
  
  /**
   * 深拷贝对象
   * @param {*} obj - 要拷贝的对象
   * @returns {*} 拷贝后的对象
   */
  deepClone(obj) {
    if (obj === null || typeof obj !== 'object') return obj;
    if (obj instanceof Date) return new Date(obj);
    if (obj instanceof Array) return obj.map(item => this.deepClone(item));
    
    const clonedObj = {};
    for (const key in obj) {
      if (obj.hasOwnProperty(key)) {
        clonedObj[key] = this.deepClone(obj[key]);
      }
    }
    return clonedObj;
  },
  
  /**
   * 清理车型名称（移除括号内容）
   * @param {string} name - 原始名称
   * @returns {string} 清理后的名称
   */
  cleanStrategyName(name) {
    if (!name) return '';
    return name.replace(/\([^)]*\)/g, '').trim();
  },
  
  /**
   * 简化车型名称（移除品牌前缀）
   * @param {string} name - 原始车型名
   * @returns {string} 简化后的名称
   */
  simplifyCarName(name) {
    if (!name) return '';
    return name.replace(/(奔驰|宝马|奥迪|大众)/g, '').trim();
  },
  
  /**
   * 复制文本到剪贴板
   * @param {string} text - 要复制的文本
   * @returns {Promise<boolean>} 是否成功
   */
  async copyToClipboard(text) {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text);
        return true;
      } else {
        // 降级方案
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        try {
          document.execCommand('copy');
          textArea.remove();
          return true;
        } catch (error) {
          console.error('复制失败:', error);
          textArea.remove();
          return false;
        }
      }
    } catch (error) {
      console.error('复制到剪贴板失败:', error);
      return false;
    }
  },
  
  /**
   * 生成CSV内容
   * @param {Array<Object>} data - 数据数组
   * @param {Array<string>} headers - 表头
   * @returns {string} CSV字符串
   */
  generateCSV(data, headers) {
    if (!data || !data.length) return '';
    
    const csvRows = [];
    csvRows.push(headers.join(','));
    
    for (const row of data) {
      const values = headers.map(header => {
        const value = row[header] || '';
        return `"${value}"`;
      });
      csvRows.push(values.join(','));
    }
    
    return csvRows.join('\n');
  },
  
  /**
   * 下载文件
   * @param {string} content - 文件内容
   * @param {string} filename - 文件名
   * @param {string} mimeType - MIME类型
   */
  downloadFile(content, filename, mimeType = 'text/plain') {
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  },
  
  /**
   * 数字动画（count-up）
   * @param {HTMLElement} element - 目标元素
   * @param {number} start - 起始值
   * @param {number} end - 结束值
   * @param {number} duration - 持续时间（毫秒）
   */
  animateNumber(element, start, end, duration = 500) {
    if (!element) return;
    
    const startTime = performance.now();
    const difference = end - start;
    
    const step = (currentTime) => {
      const elapsed = currentTime - startTime;
      const progress = Math.min(elapsed / duration, 1);
      
      // 缓动函数（ease-out）
      const easeOut = 1 - Math.pow(1 - progress, 3);
      const current = start + difference * easeOut;
      
      element.textContent = Math.round(current);
      
      if (progress < 1) {
        requestAnimationFrame(step);
      }
    };
    
    requestAnimationFrame(step);
  },
  
  /**
   * 获取环形进度条SVG路径
   * @param {number} radius - 半径
   * @param {number} strokeWidth - 线宽
   * @returns {Object} 包含circumference和dashoffset计算函数
   */
  getCircularProgress(radius, strokeWidth) {
    const normalizedRadius = radius - strokeWidth / 2;
    const circumference = normalizedRadius * 2 * Math.PI;
    
    return {
      normalizedRadius,
      circumference,
      getDashoffset(progress) {
        return circumference - (progress / 100) * circumference;
      }
    };
  }
};

// 导出Utils对象
if (typeof module !== 'undefined' && module.exports) {
  module.exports = Utils;
}
