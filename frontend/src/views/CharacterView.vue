<!-- 角色页：展示玩家属性，并以 RPG 部位格子管理当前装备与失能后补购预设。 -->
<script setup lang="ts">
import { watch } from "vue";
import { Check, EditPen, User } from "@element-plus/icons-vue";
import {
  useCharacterLoadout,
  type CharacterEmit,
  type CharacterProps,
} from "../composables/useCharacterLoadout";

const props = defineProps<CharacterProps>();
const emit = defineEmits<CharacterEmit>();
const {
  editing,
  nameDraft,
  current,
  presets,
  presetIndex,
  gridRows,
  slotDefs,
  mainAttributes,
  skills,
  proficiencies,
  openPicker,
  openConsumablePicker,
  openAmmoPicker,
  openKeyPicker,
  slotName,
  consumableSlotCount,
  consumableAt,
  ammoSlotCount,
  ammoSlotMaxRounds,
  ammoAt,
  ammoNameAt,
  ammoCellMaxRounds,
  keySlotCount,
  keySlotLabel,
  activePresetWeapon,
  activePresetAmmoOptions,
  repTag,
  liveCapacity,
  pickerOpen,
  pickerTitle,
  pickerList,
  pickOption,
  submitName,
  submitLoadout,
  hasUnsavedChanges,
} = useCharacterLoadout(props, emit);

// 未保存修改状态上报工作区，供离开角色页前的拦截提示使用。
watch(hasUnsavedChanges, (dirty) => emit("dirtyChange", dirty), { immediate: true });
</script>

<template>
  <section class="view-page character-view">
    <header class="character-identity">
      <div class="character-avatar">
        <el-icon><User /></el-icon><span class="status-dot" />
      </div>
      <div class="character-copy">
        <span class="eyebrow"
          >玩家角色 / ID {{ player.id.toString().padStart(4, "0") }}</span
        >
        <div class="character-name-row">
          <template v-if="editing">
            <el-input
              v-model="nameDraft"
              maxlength="16"
              autofocus
              @keyup.enter="submitName"
            />
            <el-button
              :icon="Check"
              :loading="savingName"
              circle
              title="保存名称"
              @click="submitName"
            />
          </template>
          <template v-else>
            <h1>{{ player.name }}</h1>
            <el-button
              :icon="EditPen"
              circle
              title="修改名称"
              @click="editing = true"
            />
          </template>
        </div>
        <p>{{ player.desc }}</p>
      </div>
      <div class="character-state">
        <span>当前状态</span>
        <strong
          :class="
            player.hp <= 0 || player.energy <= 0 || player.hydration <= 0
              ? 'text-danger'
              : 'text-success'
          "
        >
          {{
            player.hp <= 0
              ? "生命恢复中"
              : player.energy <= 0
                ? "能量恢复中"
                : player.hydration <= 0
                  ? "饮水恢复中"
                  : "状态正常"
          }}
        </strong>
      </div>
    </header>

    <section class="loadout-panel surface-panel">
      <div class="panel-heading">
        <div>
          <span>LOADOUT</span>
          <h2>装备配置</h2>
        </div>
        <small>当前装备不计入仓库容量</small>
      </div>

      <div class="loadout-area">
        <div class="loadout-block">
          <div class="loadout-block__heading">
            <strong>当前装备</strong><span>点击栏位从仓库装备</span>
          </div>
          <div class="resource-strip">
            <div class="resource-cell">
              <span>生命</span>
              <strong
                >{{ Math.round(player.hp) }}<em>/{{ player.hpMax }}</em></strong
              >
              <small>+{{ player.recoveryPerHour.hp.toFixed(1) }}/h</small>
            </div>
            <div class="resource-cell">
              <span>压力</span>
              <strong
                >{{ player.stress }}<em>/{{ Math.round(player.stressMax) }}</em></strong
              >
              <small>−{{ player.recoveryPerHour.stress.toFixed(1) }}/h</small>
            </div>
            <div class="resource-cell">
              <span>能量</span>
              <strong
                >{{ Math.round(player.energy)
                }}<em>/{{ player.energyMax }}</em></strong
              >
              <small>+{{ player.recoveryPerHour.energy.toFixed(1) }}/h</small>
            </div>
            <div class="resource-cell">
              <span>饮水</span>
              <strong
                >{{ Math.round(player.hydration)
                }}<em>/{{ player.hydrationMax }}</em></strong
              >
              <small
                >+{{ player.recoveryPerHour.hydration.toFixed(1) }}/h</small
              >
            </div>
          </div>
          <div class="slot-grid">
            <div v-for="row in gridRows" :key="row" class="slot-row">
              <button
                v-for="s in slotDefs.filter((x) => x.row === row)"
                :key="s.key"
                type="button"
                class="slot-cell"
                @click="openPicker('current', s.key)"
              >
                <span class="slot-label">{{ s.label }}</span>
                <strong :class="{ empty: !slotName('current', s.key) }">{{
                  slotName("current", s.key) || "空"
                }}</strong>
              </button>
            </div>
          </div>
          <div class="consumable-block">
            <span class="consumable-block__label">钥匙包格子 · 局外装入才开门，失能不丢</span>
            <div class="slot-row" style="flex-wrap: wrap">
              <button
                v-for="i in keySlotCount"
                :key="'key-' + i"
                type="button"
                class="slot-cell"
                @click="openKeyPicker(i - 1)"
              >
                <span class="slot-label">钥匙{{ i }}</span>
                <strong :class="{ empty: !keySlotLabel(i - 1) }">{{ keySlotLabel(i - 1) || '空' }}</strong>
              </button>
            </div>
          </div>
          <div class="consumable-block">
            <span class="consumable-block__label">随身补给</span>
            <div class="slot-row">
              <button
                v-for="i in consumableSlotCount"
                :key="i"
                type="button"
                class="slot-cell"
                @click="openConsumablePicker('current', i - 1)"
              >
                <span class="slot-label">补给{{ i }}</span>
                <strong :class="{ empty: !consumableAt(current, i - 1) }">{{
                  consumableAt(current, i - 1) || "空"
                }}</strong>
              </button>
            </div>
          </div>
          <div class="ammo-block">
            <span class="ammo-block__label">携带弹药</span>
            <span class="ammo-block__hint"
              >每格 ≤ {{ ammoSlotMaxRounds }} 发，点击选择仓库弹药</span
            >
            <div class="slot-row">
              <div v-for="i in ammoSlotCount" :key="i" class="slot-cell ammo-cell">
                <button
                  type="button"
                  class="slot-cell__pick"
                  @click="openAmmoPicker('current', i - 1)"
                >
                  <span class="slot-label">弹药{{ i }}</span>
                  <strong :class="{ empty: !ammoNameAt(current, i - 1) }">{{ ammoNameAt(current, i - 1) || "空" }}</strong>
                </button>
                <el-input-number
                  v-model="ammoAt(current, i - 1).rounds"
                  :min="0"
                  :max="ammoCellMaxRounds(ammoAt(current, i - 1).ammoId, ammoAt(current, i - 1).rounds)"
                  :disabled="!ammoAt(current, i - 1).ammoId"
                  :step="1"
                  size="small"
                  controls-position="right"
                />
              </div>
            </div>
          </div>
        </div>

        <div class="loadout-block">
          <div class="loadout-block__heading">
            <strong>预设装备</strong>
            <el-select v-model="presetIndex" size="small" class="preset-switch">
              <el-option
                v-for="(p, idx) in presets"
                :key="idx"
                :label="`预设 ${idx + 1} · ${p.name || '未命名'}`"
                :value="idx"
              />
            </el-select>
          </div>
          <div class="preset-name-row">
            <el-input
              v-model="presets[presetIndex].name"
              maxlength="10"
              placeholder="预设名称"
            />
            <small>失能丢装后按此预设补购 · 仅限商人装备</small>
          </div>
          <div class="slot-grid">
            <div v-for="row in gridRows" :key="row" class="slot-row">
              <button
                v-for="s in slotDefs.filter((x) => x.row === row)"
                :key="s.key"
                type="button"
                class="slot-cell"
                @click="openPicker('preset', s.key)"
              >
                <span class="slot-label">{{ s.label }}</span>
                <strong :class="{ empty: !slotName('preset', s.key) }">{{
                  slotName("preset", s.key) || "空"
                }}</strong>
              </button>
            </div>
          </div>
          <div v-if="activePresetWeapon?.ammoPerRound" class="preset-name-row">
            <el-select
              v-model="presets[presetIndex].ammoId"
              :placeholder="activePresetAmmoOptions.length ? '选择兼容弹药' : '暂无武器商人可售弹药'"
              :disabled="!activePresetAmmoOptions.length"
            >
              <el-option
                v-for="ammo in activePresetAmmoOptions"
                :key="ammo.id"
                :label="`${ammo.name}${repTag(ammo.repRequirement)}`"
                :value="ammo.id"
              />
            </el-select>
            <el-input-number
              v-model="presets[presetIndex].ammoRounds"
              :min="activePresetWeapon.ammoPerRound"
              :max="9999"
              :step="activePresetWeapon.ammoPerRound"
              controls-position="right"
            />
            <small
              >失能恢复时优先取仓库中的预设弹药；不足一次攻击时，自动购买当时可用的最高等级弹药</small
            >
          </div>
          <div class="consumable-block">
            <span class="consumable-block__label">预设补给</span>
            <div class="slot-row">
              <button
                v-for="i in consumableSlotCount"
                :key="i"
                type="button"
                class="slot-cell"
                @click="openConsumablePicker('preset', i - 1)"
              >
                <span class="slot-label">补给{{ i }}</span>
                <strong
                  :class="{ empty: !consumableAt(presets[presetIndex], i - 1) }"
                  >{{
                    consumableAt(presets[presetIndex], i - 1) || "空"
                  }}</strong
                >
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="carry-strip">
        <span
          >可携带格数
          <b
            >{{ liveCapacity.usedSlots }} / {{ liveCapacity.totalSlots }}</b
          ></span
        >
        <span
          >可携带负重
          <b
            >{{ liveCapacity.usedWeight.toFixed(1) }} /
            {{ liveCapacity.totalWeight.toFixed(1) }} kg</b
          ></span
        >
        <small
          >基础 {{ liveCapacity.baseWeight.toFixed(1) }}kg + 胸挂/背包加成
          {{ liveCapacity.bonusWeight.toFixed(1) }}kg · 安全箱 +{{ liveCapacity.secureSlots }} 格（箱内计入负重）</small
        >
      </div>
      <div class="loadout-actions">
        <span>预设补购受现金限制，且不计入仓库容量</span
        ><el-button
          type="primary"
          :loading="savingLoadout"
          @click="submitLoadout"
          >保存装备配置</el-button
        >
      </div>
    </section>

    <div class="attribute-strip">
      <div
        v-for="attr in mainAttributes"
        :key="attr.label"
        class="attribute-cell"
      >
        <span>{{ attr.code }}</span
        ><strong>{{ attr.value }}</strong
        ><small>{{ attr.label }}</small>
      </div>
    </div>

    <div class="character-grid">
      <section class="surface-panel skill-section">
        <div class="panel-heading">
          <div>
            <span>SKILL</span>
            <h2>生存技能</h2>
          </div>
          <small>行动成功判定会缓慢提升</small>
        </div>
        <div class="progress-list">
          <div v-for="skill in skills" :key="skill[0]" class="progress-row">
            <span>{{ skill[0] }}</span
            ><el-progress :percentage="skill[1]" :show-text="false" /><b>{{
              skill[1]
            }}</b>
          </div>
        </div>
      </section>
      <section class="surface-panel skill-section">
        <div class="panel-heading">
          <div>
            <span>PROF</span>
            <h2>武器熟练度</h2>
          </div>
          <small>撤离成功且有效使用后提升</small>
        </div>
        <div class="progress-list">
          <div
            v-for="prof in proficiencies"
            :key="prof[0]"
            class="progress-row"
          >
            <span>{{ prof[0] }}</span
            ><el-progress :percentage="prof[1]" :show-text="false" /><b>{{
              prof[1]
            }}</b>
          </div>
        </div>
      </section>
    </div>

    <el-dialog v-model="pickerOpen" :title="pickerTitle" width="440px">
      <div class="slot-picker-list">
        <button type="button" class="slot-picker-item" @click="pickOption('')">
          <span class="slot-picker-item__name">不装备</span
          ><small>清除该栏位</small>
        </button>
        <button
          v-for="item in pickerList"
          :key="item.id"
          type="button"
          class="slot-picker-item"
          @click="pickOption(item.id)"
        >
          <span class="slot-picker-item__name">{{ item.name }}</span
          ><small>{{ item.detail }}</small>
        </button>
      </div>
    </el-dialog>
  </section>
</template>
