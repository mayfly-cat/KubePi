<template xmlns:el-col="http://www.w3.org/1999/html">
    <div class="sso-background">
      <div class="sso-container">
        <el-row type="flex" v-loading="loading">
          <el-col :span="24">
            <el-form size="default">
              <div class="sso-title">
                {{ systemName }}
              </div>
            <div class="sso-tips">
                {{ countdownMessage }}
              </div>
              <div class="sso-btn">
                <el-button class="spacing-button" type="primary" @click="confirm" size="default">
                  {{ $t("commons.button.confirm") }}
                </el-button>
                <el-button class="spacing-button" @click="cancel" size="default">
                  {{ $t("commons.button.cancel") }}
                </el-button>
              </div>
            </el-form>
          </el-col>
        </el-row>
      </div>
    </div>
</template>

<script>
import { ssoLogin,getSso } from "@/api/sso"
export default {
  name: "Login",
  data () {
    return {
      loading: false,
      countdown: 3,
      intervalId: null,
      redirect: undefined,
      systemName: this.$t("commons.sso.title"),
      loadingPage: false,
      authType: "",
    }
  },
  computed: {
    countdownMessage() {
      if (this.countdown > 0) {
        return `正在跳转${this.authType}认证，${this.countdown}...`;
      } else {
        return '已完成跳转，准备进入...';
      }
    }
  },
  methods: {
    redirectSso () {
      clearInterval(this.intervalId)
      window.location.href = '/gvp'+ssoLogin()
      console.log("重定向完成")
      // 这里可以添加重定向逻辑，例如：
      //window.location.href = "https://www.google.com.hk/";
    },
    cancel () {
      clearInterval(this.intervalId)
      window.location.href = '/gvp/login'
    },
    confirm () {
      clearInterval(this.intervalId)
      window.location.href = '/gvp'+ssoLogin()
    },
    getAuthType() {
      getSso().then((res) => {
        if (res.data.protocol == "openid") {
          this.authType = " OpenID "
        } else {
          this.authType = " SAML2 "
        }
      }).catch(error => {
        console.error('Failed to fetch auth type:', error);
        this.authType = 'OpenID';
      });
    },
  },
  mounted() {
    this.getAuthType();
    this.intervalId = setInterval(() => {
      if (this.countdown > 0) {
        this.countdown -= 1;
      } else {
        clearInterval(this.intervalId);
        this.redirectSso();
      }
    }, 1000);
  },
  beforeDestroy() {
    if (this.intervalId) {
      clearInterval(this.intervalId);
    }
  }
}
</script>

<style lang="scss" scoped>
  @import "../../styles/common/variables";
  @mixin sso-center {
    display: flex;
    justify-content: center;
    align-items: center;
  }
  .sso-background {
    background-color: #F5F7FA;
    height: 100%;
    @include sso-center;
  }
  .sso-container {
    min-width: 420px;
    width: 520px;
    padding: 32px 40px 28px;
    background-color: #FFFFFF;
    box-shadow: 0 5px 5px -3px rgba(0, 0, 0, .2), 0 8px 10px 1px rgba(0, 0, 0, .14), 0 3px 14px 2px rgba(0, 0, 0, .12);
    @media only screen and (max-width: 1280px) {
      width: 420px;
      min-width: 320px;
    }
    .sso-title {
      margin-top: 4px;
      font-size: 32px;
      letter-spacing: 0;
      text-align: center;
      color: #303133;
      @media only screen and (max-width: 1280px) {
        margin-top: 4px;
      }
    }
    .sso-tips {
      margin-top: 24px;
      font-size: 18px;
      letter-spacing: 0;
      text-align: center;
      color: #606266;
      @media only screen and (max-width: 1280px) {
        margin-top: 16px;
      }
    }
    .sso-btn {
      margin-top: 32px;
      text-align: center;
      @media only screen and (max-width: 1280px) {
        margin-top: 24px;
      }
    }
    .sso-image {
      background: url(../../assets/KubePi-login.jpg) no-repeat;
      background-size: cover;
      width: 100%;
      height: 280px;
      @media only screen and (max-width: 1280px) {
        height: 380px;
      }
    }
    .submit {
      width: 100%;
      border-radius: 0;
    }
    .spacing-button + .spacing-button {
      margin-left: 16px;
      background-color: transparent;
      color: black;
    }
  }
</style>