import {describe, expect, it} from 'vitest';
import type {ConfigDTO} from '../types';
import {configFormUpdates, ttsVoiceForEngine} from './configForms';
const cfg = {pipeline:{workers:3,tts_workers:4,max_speed_factor:0,whisper_mode:'openai',whisper_bin:'/whisper',whisper_model:'/model',asr_base_url:'https://asr.test',asr_api_key:'asr-key',asr_model:'asr',default_output_dir:'/output',default_target_lang:'ja',translate_chunk_size:7,translate_prompt:'术语',ffmpeg_bin:'/ffmpeg',clean_intermediate:true},llm:{provider:'bing',base_url:'https://llm.test',api_key:'llm-key',model:'llm'},tts:{type:'edge',voice:'custom-openai',base_url:'https://tts.test',api_key:'tts-key',model:'custom-tts'}} as ConfigDTO;
describe('配置页保存范围及音色保留',()=>{
 it('保存 OpenAI 配置不会把默认 Edge 音色写到 OpenAI 输入框',()=>{
  const updates=configFormUpdates('tts',cfg,'openai');
  expect(updates.tts).toMatchObject({type:'edge',openai_voice:'custom-openai',voice:'zh-CN-XiaoxiaoNeural',model:'custom-tts'});
  expect(ttsVoiceForEngine('openai',updates.tts!)).toBe('custom-openai');
  expect(updates.pipeline).toEqual({tts_workers:4,max_speed_factor:0});
  expect(updates.llm).toBeUndefined();
 });
 it('设为默认只构造提交值，保留旧配置；兼容旧版音色',()=>{
  const updates=configFormUpdates('tts',cfg,'openai','openai');
  expect(updates.tts).toMatchObject({type:'openai',voice:'custom-openai'});
  expect(cfg.tts.type).toBe('edge');
  expect(ttsVoiceForEngine('openai',{...cfg.tts,type:'OpenAI'})).toBe('custom-openai');
  expect(ttsVoiceForEngine('edge',{...cfg.tts,type:'',voice:'en-US-JennyNeural'})).toBe('en-US-JennyNeural');
 });
 it('分别保存 Edge 和 OpenAI 音色，切回时取各自音色',()=>{
  const first=configFormUpdates('tts',cfg,'openai').tts!;
  const second=configFormUpdates('tts',{...cfg,tts:{...first,voice:'en-US-JennyNeural'}},'edge').tts!;
  expect(ttsVoiceForEngine('openai',second)).toBe('custom-openai');
  expect(ttsVoiceForEngine('edge',second)).toBe('en-US-JennyNeural');
 });
 it('识别、翻译、全局设置只提交所属表单的字段',()=>{
  expect(configFormUpdates('asr',cfg).pipeline).toMatchObject({whisper_bin:'/whisper',asr_model:'asr'});
  expect(configFormUpdates('asr',cfg).tts).toBeUndefined();
  expect(configFormUpdates('translation',cfg).pipeline).toEqual({default_target_lang:'ja',translate_chunk_size:7,translate_prompt:'术语'});
  expect(configFormUpdates('translation',cfg).tts).toBeUndefined();
  expect(configFormUpdates('settings',cfg).pipeline).toEqual({workers:3,ffmpeg_bin:'/ffmpeg',default_output_dir:'/output',default_target_lang:'ja',clean_intermediate:true});
  expect(configFormUpdates('settings',cfg).llm).toBeUndefined();
 });
});
