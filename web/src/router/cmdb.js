import Host from '@/views/cmdb/cmdbHost.vue'
import Group from '@/views/cmdb/cmdbGroup.vue'
import Db from '@/views/cmdb/cmdbDB.vue'
import SSH from '@/views/cmdb/Host/SSH.vue'
import DBdetails from '@/views/cmdb/DBdetails.vue'
import Switch from '@/views/cmdb/cmdbSwitch.vue'


const routes = [
    {
        path: '/cmdb/ecs',
        component: Host,
        meta: {sTitle: '资产管理', tTitle: '主机管理'}
    },
    {
        path: '/cmdb/group',
        component: Group,
        meta: {sTitle: '资产管理', tTitle: '业务分组'}
    },
    {
        path: '/cmdb/db',
        component: Db,
        meta: {sTitle: '资产管理', tTitle: '数据管理'}
    }, 
    {
        path: '/cmdb/ssh',
        component: SSH,
        meta: {sTitle: '资产管理', tTitle: '终端登录'}
    },
    {
        path: '/cmdb/dbdetails',
        component: DBdetails,
        meta: {sTitle: '数据管理', tTitle: '数据库操作'}
    },
    {
        path: '/cmdb/switch',
        component: Switch,
        meta: {sTitle: '资产管理', tTitle: '网络设备'}
    }

]

export default routes
