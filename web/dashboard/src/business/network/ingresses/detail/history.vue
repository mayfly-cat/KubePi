<template>
  <div>
    <el-table :data="historyList" v-loading="loading" style="width: 100%">
      <el-table-column :label="$t('business.ingress.history_version')" prop="version" width="100">
      </el-table-column>
      <el-table-column :label="$t('business.ingress.history_operator')" prop="operator" width="150">
      </el-table-column>
      <el-table-column :label="$t('business.ingress.history_description')" prop="description" show-overflow-tooltip>
      </el-table-column>
      <el-table-column :label="$t('commons.table.created_time')" prop="createAt" width="180">
        <template v-slot:default="{row}">
          <span v-if="row.createAt">{{ new Date(row.createAt).toLocaleString() }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('commons.table.action')" width="200" fixed="right">
        <template v-slot:default="{row}">
          <el-button size="mini" @click="viewHistory(row)">{{ $t('commons.button.detail') }}</el-button>
          <el-button 
            size="mini" 
            type="primary" 
            @click="rollbackHistory(row)"
            :disabled="row.version === currentVersion"
          >
            {{ $t('business.ingress.rollback') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 历史版本详情对话框 -->
    <el-dialog
      :title="$t('business.ingress.history_detail')"
      :visible.sync="historyDialogVisible"
      width="80%"
    >
      <yaml-editor v-if="selectedHistory" :value="selectedHistoryData" :read-only="true"></yaml-editor>
      <span slot="footer" class="dialog-footer">
        <el-button @click="historyDialogVisible = false">{{ $t('commons.button.cancel') }}</el-button>
        <el-button 
          type="primary" 
          @click="rollbackFromDialog"
          :disabled="selectedHistory && selectedHistory.version === currentVersion"
        >
          {{ $t('business.ingress.rollback') }}
        </el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { listIngressHistory, getIngressHistory, rollbackIngress } from "@/api/ingress-history"
import { updateIngress } from "@/api/ingress"
import YamlEditor from "@/components/yaml-editor"

export default {
  name: "IngressHistory",
  components: { YamlEditor },
  props: {
    cluster: String,
    namespace: String,
    ingressName: String
  },
  data() {
    return {
      historyList: [],
      loading: false,
      historyDialogVisible: false,
      selectedHistory: null,
      selectedHistoryData: null,
      currentVersion: 0
    }
  },
  methods: {
    async loadHistory() {
      if (!this.cluster || !this.namespace || !this.ingressName) {
        return
      }
      this.loading = true
      try {
        const res = await listIngressHistory(this.cluster, this.namespace, this.ingressName)
        this.historyList = res.data || []
        // 获取当前版本号（最新版本）
        if (this.historyList.length > 0) {
          this.currentVersion = this.historyList[0].version
        }
      } catch (error) {
        this.$message({
          type: "error",
          message: this.$t("business.ingress.load_history_failed") + ": " + (error.response?.data?.message || error.message)
        })
      } finally {
        this.loading = false
      }
    },
    async viewHistory(row) {
      this.loading = true
      try {
        const res = await getIngressHistory(this.cluster, this.namespace, this.ingressName, row.version)
        this.selectedHistory = row
        this.selectedHistoryData = res.data.ingressData
        this.historyDialogVisible = true
      } catch (error) {
        this.$message({
          type: "error",
          message: this.$t("business.ingress.load_history_detail_failed") + ": " + (error.response?.data?.message || error.message)
        })
      } finally {
        this.loading = false
      }
    },
    rollbackHistory(row) {
      this.$confirm(
        this.$t("business.ingress.rollback_confirm", { version: row.version }),
        this.$t("commons.message_box.prompt"),
        {
          confirmButtonText: this.$t("commons.button.confirm"),
          cancelButtonText: this.$t("commons.button.cancel"),
          type: "warning",
        }
      ).then(() => {
        this.doRollback(row.version)
      })
    },
    rollbackFromDialog() {
      if (!this.selectedHistory) return
      this.rollbackHistory(this.selectedHistory)
    },
    async doRollback(version) {
      this.loading = true
      try {
        // 获取回滚数据
        const res = await rollbackIngress(this.cluster, this.namespace, this.ingressName, version)
        const ingressData = res.data

        // 更新 Ingress（这会自动保存新的历史版本）
        await updateIngress(this.cluster, this.namespace, this.ingressName, ingressData)

        this.$message({
          type: "success",
          message: this.$t("business.ingress.rollback_success")
        })

        // 重新加载历史列表
        await this.loadHistory()
        this.historyDialogVisible = false
      } catch (error) {
        this.$message({
          type: "error",
          message: this.$t("business.ingress.rollback_failed") + ": " + (error.response?.data?.message || error.message)
        })
      } finally {
        this.loading = false
      }
    }
  },
  watch: {
    cluster: {
      handler() {
        this.loadHistory()
      },
      immediate: true
    },
    namespace: {
      handler() {
        this.loadHistory()
      },
      immediate: true
    },
    ingressName: {
      handler() {
        this.loadHistory()
      },
      immediate: true
    }
  }
}
</script>

<style scoped>
</style>

