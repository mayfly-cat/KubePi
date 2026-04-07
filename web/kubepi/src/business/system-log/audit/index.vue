<template>
  <layout-content :header="$t('business.system.audit_log')">
    <div v-loading="loading">
      <complex-table :search-config="searchConfig" :data="data" :pagination-config="paginationConfig" @search="search">
        <el-table-column :label="$t('business.cluster.cluster')" prop="cluster" fix />
        <el-table-column :label="$t('business.system.operator')" prop="operator" fix />
        <el-table-column :label="$t('business.system.http_method')" prop="httpMethod" fix />
        <el-table-column :label="$t('business.system.request_path')" prop="requestPath" min-width="200" show-overflow-tooltip />
        <el-table-column :label="$t('business.system.resource')" prop="resource" fix />
        <el-table-column :label="$t('business.system.operation')" prop="operation" fix>
          <template v-slot:default="{row}">
            {{ translate(row.operation) }}
          </template>
        </el-table-column>
        <el-table-column :label="$t('business.system.operation_domain')" prop="operationDomain" fix>
          <template v-slot:default="{row}">
            {{ translate(row.operationDomain) }}
          </template>
        </el-table-column>
        <el-table-column :label="$t('business.system.specific_information')" prop="specificInformation" min-width="120" show-overflow-tooltip />
        <el-table-column :label="$t('business.system.client_ip')" prop="clientIp" fix />
        <el-table-column :label="$t('business.system.status_code')" prop="statusCode" width="90" fix />
        <el-table-column :label="$t('business.system.audit_result')" prop="success" width="90" fix>
          <template v-slot:default="{row}">
            <el-tag v-if="row.success" type="success" size="small">{{ $t("business.system.result_ok") }}</el-tag>
            <el-tag v-else type="danger" size="small">{{ $t("business.system.result_fail") }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('commons.table.created_time')" fix width="168">
          <template v-slot:default="{row}">
            {{ row.createAt | datetimeFormat }}
          </template>
        </el-table-column>
      </complex-table>
    </div>
  </layout-content>
</template>

<script>
import LayoutContent from "@/components/layout/LayoutContent"
import ComplexTable from "@/components/complex-table"
import { searchAuditLogs } from "@/api/systems"

export default {
  name: "AuditLog",
  components: { ComplexTable, LayoutContent },
  data() {
    return {
      paginationConfig: {
        currentPage: 1,
        pageSize: 10,
        total: 0,
      },
      loading: false,
      searchConfig: {
        quickPlaceholder: this.$t("commons.search.quickSearch"),
        components: [
          {
            field: "cluster",
            label: this.$t("business.cluster.cluster"),
            component: "FuComplexInput",
            defaultOperator: "eq",
          },
          {
            field: "operator",
            label: this.$t("business.system.operator"),
            component: "FuComplexInput",
            defaultOperator: "eq",
          },
          {
            field: "httpMethod",
            label: this.$t("business.system.http_method"),
            component: "FuComplexInput",
            defaultOperator: "eq",
          },
          {
            field: "resource",
            label: this.$t("business.system.resource"),
            component: "FuComplexInput",
            defaultOperator: "eq",
          },
          {
            field: "success",
            label: this.$t("business.system.audit_result"),
            component: "FuComplexSelect",
            defaultOperator: "eq",
            options: [
              { label: this.$t("business.system.result_ok"), value: "true" },
              { label: this.$t("business.system.result_fail"), value: "false" },
            ],
          },
        ],
      },
      data: [],
    }
  },
  methods: {
    translate(a) {
      return this.$t(a)
    },
    search(conditions) {
      this.loading = true
      const { currentPage, pageSize } = this.paginationConfig
      searchAuditLogs(currentPage, pageSize, conditions).then((data) => {
        this.loading = false
        this.data = data.data.items
        this.paginationConfig.total = data.data.total
      })
    },
  },
  created() {
    this.search()
  },
}
</script>

<style scoped>
</style>
