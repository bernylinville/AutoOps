<template>
  <div class="code-editor">
    <textarea
      ref="textarea"
      :value="modelValue"
      @input="handleInput"
      @scroll="syncScroll"
      @keydown="handleKeydown"
      @click="syncScroll"
      @focus="syncScroll"
      class="code-editor__textarea"
      :style="{ height, fontSize }"
      :readonly="readonly"
    />
    <pre 
      ref="highlightRef"
      class="code-editor__highlight"
      v-html="highlightedCode"
      :style="{ height, fontSize }"
      @click="focusEditor"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { highlight } from '@/utils/highlight'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  language: {
    type: String,
    default: null
  },
  height: {
    type: String,
    default: '300px'
  },
  readonly: {
    type: Boolean,
    default: false
  },
  fontSize: {
    type: String,
    default: '14px'
  },
  searchText: {
    type: String,
    default: ''
  },
  searchResults: {
    type: Array,
    default: () => []
  },
  currentSearchIndex: {
    type: Number,
    default: -1
  }
})

const emit = defineEmits(['update:modelValue'])

  const textarea = ref(null)
  const highlightRef = ref(null)

  // 确保编辑器能正确聚焦
  const focusEditor = () => {
    if (textarea.value && !props.readonly) {
      textarea.value.focus()
    }
  }

  const highlightedCode = computed(() => {
    let content = props.modelValue || ''
    let highlightedContent = highlight(content, props.language)

    // 确保内容以换行符结尾，以匹配textarea的行为
    if (content && !content.endsWith('\n')) {
      highlightedContent += '\n'
    }

    // 如果有搜索文本，添加搜索高亮
    if (props.searchText && props.searchText.trim()) {
      const searchText = props.searchText.toLowerCase()
      const lines = highlightedContent.split('\n')

      highlightedContent = lines.map((line, lineIndex) => {
        let processedLine = line
        const lowerLine = line.toLowerCase()
        let offset = 0

        // 查找该行中的所有匹配项
        let startIndex = 0
        // eslint-disable-next-line no-constant-condition
        while (true) {
          const matchIndex = lowerLine.indexOf(searchText, startIndex)
          if (matchIndex === -1) break

          // 检查是否是当前搜索结果
          const isCurrentResult = props.searchResults.some((result, index) => {
            return result.lineIndex === lineIndex &&
                   result.matchIndex === matchIndex &&
                   index === props.currentSearchIndex
          })

          // 应用高亮样式
          const before = processedLine.substring(0, matchIndex + offset)
          const match = processedLine.substring(matchIndex + offset, matchIndex + offset + searchText.length)
          const after = processedLine.substring(matchIndex + offset + searchText.length)

          const highlightClass = isCurrentResult ? 'search-highlight-current' : 'search-highlight'
          processedLine = before + `<span class="${highlightClass}">${match}</span>` + after

          // 更新偏移量（因为添加了HTML标签）
          const addedLength = `<span class="${highlightClass}"></span>`.length
          offset += addedLength
          startIndex = matchIndex + searchText.length
        }

        return processedLine
      }).join('\n')
    }

    return highlightedContent
  })

  const handleInput = (e) => {
    if (!props.readonly) {
      emit('update:modelValue', e.target.value)
      // 输入时立即同步滚动
      setTimeout(() => {
        syncScroll()
      }, 0)
    }
  }

  const handleKeydown = (e) => {
    // 在按键后同步滚动位置（特别是方向键、回车、删除等）
    setTimeout(() => {
      syncScroll()
    }, 0)
  }

  const syncScroll = () => {
    if (highlightRef.value && textarea.value) {
      // 在滚动时也检查并同步高度
      const textareaScrollHeight = textarea.value.scrollHeight
      const highlightScrollHeight = highlightRef.value.scrollHeight

      if (Math.abs(textareaScrollHeight - highlightScrollHeight) > 2) {
        highlightRef.value.style.height = textareaScrollHeight + 'px'
        highlightRef.value.style.minHeight = textareaScrollHeight + 'px'
      }

      // 精确同步滚动位置
      const targetScrollTop = textarea.value.scrollTop
      const targetScrollLeft = textarea.value.scrollLeft

      highlightRef.value.scrollTop = targetScrollTop
      highlightRef.value.scrollLeft = targetScrollLeft

      // 二次确认滚动位置，特别是在底部区域
      setTimeout(() => {
        if (highlightRef.value && textarea.value) {
          if (Math.abs(highlightRef.value.scrollTop - textarea.value.scrollTop) > 1) {
            highlightRef.value.scrollTop = textarea.value.scrollTop
          }
          if (Math.abs(highlightRef.value.scrollLeft - textarea.value.scrollLeft) > 1) {
            highlightRef.value.scrollLeft = textarea.value.scrollLeft
          }
        }
      }, 1)
    }
  }

  // 同步所有样式属性和尺寸
  const syncStyles = () => {
    if (highlightRef.value && textarea.value) {
      // 强制同步关键样式属性
      const textareaStyles = window.getComputedStyle(textarea.value)
      const highlight = highlightRef.value

      // 确保字体相关属性完全一致
      highlight.style.fontSize = textareaStyles.fontSize
      highlight.style.fontFamily = textareaStyles.fontFamily
      highlight.style.lineHeight = textareaStyles.lineHeight
      highlight.style.letterSpacing = textareaStyles.letterSpacing
      highlight.style.wordSpacing = textareaStyles.wordSpacing
      highlight.style.tabSize = textareaStyles.tabSize

      // 同步尺寸和布局相关属性
      highlight.style.width = textareaStyles.width
      highlight.style.height = textareaStyles.height
      highlight.style.padding = textareaStyles.padding
      highlight.style.margin = textareaStyles.margin
      highlight.style.boxSizing = textareaStyles.boxSizing

      // 强制同步scrollHeight - 这是关键的修复
      const textareaScrollHeight = textarea.value.scrollHeight
      const highlightScrollHeight = highlight.scrollHeight

      if (Math.abs(textareaScrollHeight - highlightScrollHeight) > 2) {
        // 如果scrollHeight差异大于2px，强制调整highlight的高度
        highlight.style.height = textareaScrollHeight + 'px'
        highlight.style.minHeight = textareaScrollHeight + 'px'
      }

      // 确保内容高度一致
      const textareaRect = textarea.value.getBoundingClientRect()
      const highlightRect = highlight.getBoundingClientRect()

      if (Math.abs(textareaRect.height - highlightRect.height) > 1) {
        highlight.style.minHeight = textareaRect.height + 'px'
      }

      // 同步滚动位置
      syncScroll()
    }
  }

onMounted(() => {
  // 初始化时同步样式和滚动
  setTimeout(() => {
    syncStyles()
  }, 100) // 延迟一点确保DOM完全渲染

  // 监听textarea尺寸变化
  if (textarea.value && window.ResizeObserver) {
    const resizeObserver = new ResizeObserver(() => {
      syncStyles()
    })
    resizeObserver.observe(textarea.value)

    // 清理函数
    onUnmounted(() => {
      resizeObserver.disconnect()
    })
  }
})

watch(() => props.modelValue, () => {
  // 内容变化时同步滚动和样式
  setTimeout(() => {
    syncStyles()
  }, 10)
})

// 导出方法供父组件使用
defineExpose({
  focus: focusEditor
})
</script>

<style scoped>
.code-editor {
  position: relative;
  width: 100%;
  overflow: hidden;
  border-radius: 4px;
  border: 1px solid #dcdfe6;
}

.code-editor__textarea,
.code-editor__highlight {
  margin: 0;
  padding: 10px;
  font-family: Consolas, Monaco, 'Andale Mono', monospace;
  font-size: 14px;
  line-height: 1.5;
  letter-spacing: 0;
  tab-size: 2;
  white-space: pre;
  word-wrap: normal;
  word-break: normal;
  width: 100%;
  min-height: 100%;
  box-sizing: border-box;
  background: transparent;
  border: none;
  outline: none;
  vertical-align: top;
}

.code-editor__textarea {
  position: absolute;
  top: 0;
  left: 0;
  color: transparent;
  background: rgba(255,255,255,0.01);
  caret-color: #409EFF;
  caret-width: 2px;
  resize: none;
  outline: none;
  border: none;
  overflow: hidden;
  overflow-wrap: normal;
  z-index: 1;
}

.code-editor__textarea:focus {
  caret-color: #67C23A;
  background: rgba(103, 194, 58, 0.05);
}

.code-editor__highlight {
  position: relative;
  pointer-events: none;
  overflow: hidden;
  overflow-wrap: normal;
  z-index: 0;
  background: #1e1e1e;
  color: #d4d4d4;
  /* 强制与textarea完全一致的高度计算 */
  display: block;
  white-space: pre;
}

/* 搜索高亮样式 */
:deep(.search-highlight) {
  background-color: #ffd700;
  color: #000;
  padding: 1px 2px;
  border-radius: 2px;
}

:deep(.search-highlight-current) {
  background-color: #ff6b35;
  color: #fff;
  padding: 1px 2px;
  border-radius: 2px;
  box-shadow: 0 0 3px rgba(255, 107, 53, 0.5);
}
</style>
