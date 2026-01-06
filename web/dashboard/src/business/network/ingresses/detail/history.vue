<template>
  <div>
    <div v-if="!loading && historyList.length === 0" style="text-align: center; padding: 40px; color: #909399;">
      <i class="el-icon-info" style="font-size: 48px; margin-bottom: 16px;"></i>
      <p>{{ $t('business.ingress.no_history_version') || '暂无历史版本记录' }}</p>
    </div>
    <el-table :data="historyList" v-loading="loading" style="width: 100%" v-else>
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
            :disabled="isRollbackDisabled(row)"
            :title="getRollbackTooltip(row)"
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
          :disabled="selectedHistory && isRollbackDisabled(selectedHistory)"
          :title="selectedHistory ? getRollbackTooltip(selectedHistory) : ''"
        >
          {{ $t('business.ingress.rollback') }}
        </el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { listIngressHistory, getIngressHistory, rollbackIngress } from "@/api/ingress-history"
import { updateIngress, getIngress } from "@/api/ingress"
import YamlEditor from "@/components/yaml-editor"
import { checkPermissions } from "@/utils/permission"

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
      currentVersion: 0,
      hasUpdatePermission: false
    }
  },
  methods: {
    // 检查是否有更新权限
    checkUpdatePermission() {
      this.hasUpdatePermission = checkPermissions({
        scope: "namespace",
        apiGroup: "networking.k8s.io",
        resource: "ingresses",
        verb: "update",
      })
    },
    // 判断回滚按钮是否应该禁用
    isRollbackDisabled(row) {
      // 如果是当前版本，禁用
      if (row.version === this.currentVersion) {
        return true
      }
      // 如果没有更新权限，禁用
      if (!this.hasUpdatePermission) {
        return true
      }
      return false
    },
    // 获取回滚按钮的提示信息
    getRollbackTooltip(row) {
      if (row.version === this.currentVersion) {
        return this.$t("business.ingress.rollback_disabled_current_version") || "当前版本，无法回滚"
      }
      if (!this.hasUpdatePermission) {
        return this.$t("business.ingress.rollback_disabled_no_permission") || "没有更新权限，无法回滚"
      }
      return this.$t("business.ingress.rollback_tooltip", { version: row.version }) || `回滚到版本 ${row.version}`
    },
    async loadHistory() {
      if (!this.cluster || !this.namespace || !this.ingressName) {
        return
      }
      this.loading = true
      try {
        const res = await listIngressHistory(this.cluster, this.namespace, this.ingressName)
        this.historyList = res.data || []
        // 获取当前版本号（最新版本，版本号最大的）
        if (this.historyList.length > 0) {
          // 确保按版本号降序排列（最新版本在前）
          this.historyList.sort((a, b) => (b.version || 0) - (a.version || 0))
          // 当前版本是版本号最大的那个
          this.currentVersion = this.historyList[0].version
        } else {
          // 如果没有历史版本，当前版本为 0
          this.currentVersion = 0
        }
        // 检查权限
        this.checkUpdatePermission()
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
      // 检查权限
      if (!this.hasUpdatePermission) {
        this.$message({
          type: "warning",
          message: this.$t("business.ingress.rollback_disabled_no_permission") || "没有更新权限，无法执行回滚操作"
        })
        return
      }
      
      // 检查是否是当前版本
      if (row.version === this.currentVersion) {
        this.$message({
          type: "info",
          message: this.$t("business.ingress.rollback_disabled_current_version") || "当前版本，无需回滚"
        })
        return
      }
      
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
      }).catch(() => {
        // 用户取消操作
      })
    },
    rollbackFromDialog() {
      if (!this.selectedHistory) return
      this.rollbackHistory(this.selectedHistory)
    },
    async doRollback(version) {
      this.loading = true
      try {
        // 先获取最新的 Ingress（包含最新的 resourceVersion 和其他元数据）
        const currentIngressRes = await getIngress(this.cluster, this.namespace, this.ingressName)
        const currentIngress = currentIngressRes

        // 获取历史版本数据
        const res = await rollbackIngress(this.cluster, this.namespace, this.ingressName, version)
        const historyIngressData = res.data

        // 合并历史版本的配置到最新的 Ingress
        // 只更新 spec，保留所有最新的 metadata（resourceVersion, uid, creationTimestamp, generation 等）
        const rollbackData = {
          apiVersion: currentIngress.apiVersion || historyIngressData.apiVersion,
          kind: currentIngress.kind || historyIngressData.kind,
          metadata: {
            ...currentIngress.metadata, // 保留最新的 metadata（包括 resourceVersion）
            // 可选：如果需要保留历史版本的 annotations 和 labels，可以取消下面的注释
            // annotations: historyIngressData.metadata?.annotations || currentIngress.metadata.annotations,
            // labels: historyIngressData.metadata?.labels || currentIngress.metadata.labels,
          },
          spec: historyIngressData.spec || currentIngress.spec
        }

        // 更新 Ingress（这会自动保存新的历史版本）
        await updateIngress(this.cluster, this.namespace, this.ingressName, rollbackData)

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

