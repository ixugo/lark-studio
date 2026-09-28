export type TtsEngine = 'edge' | 'openai';

export interface TtsVoiceOption {
  value: string;
  zhLabel: string;
  enLabel: string;
}

export const DEFAULT_EDGE_TTS_VOICE = 'zh-CN-YunjianNeural';
export const DEFAULT_OPENAI_TTS_VOICE = 'alloy';

// 工作台与语音合成页共用此清单，避免两处音色漂移。
export const EDGE_TTS_VOICES: TtsVoiceOption[] = [
  { value: 'zh-CN-YunjianNeural', zhLabel: '云健 · 普通话男声 · 热情有力', enLabel: 'Yunjian · Mandarin male · Passionate' },
  { value: 'zh-CN-XiaoxiaoNeural', zhLabel: '晓晓 · 普通话女声 · 温暖自然', enLabel: 'Xiaoxiao · Mandarin female · Warm' },
  { value: 'zh-CN-YunxiNeural', zhLabel: '云希 · 普通话男声 · 阳光活泼', enLabel: 'Yunxi · Mandarin male · Bright' },
  { value: 'zh-CN-YunxiaNeural', zhLabel: '云夏 · 普通话男声 · 少年感', enLabel: 'Yunxia · Mandarin male · Youthful' },
  { value: 'zh-CN-XiaoyiNeural', zhLabel: '晓伊 · 普通话女声 · 活泼灵动', enLabel: 'Xiaoyi · Mandarin female · Lively' },
  { value: 'zh-CN-YunyangNeural', zhLabel: '云扬 · 普通话男声 · 新闻播报', enLabel: 'Yunyang · Mandarin male · News' },
  { value: 'zh-CN-liaoning-XiaobeiNeural', zhLabel: '晓北 · 辽宁方言女声 · 幽默', enLabel: 'Xiaobei · Liaoning female · Humorous' },
  { value: 'zh-CN-shaanxi-XiaoniNeural', zhLabel: '晓妮 · 陕西方言女声 · 明亮', enLabel: 'Xiaoni · Shaanxi female · Bright' },
  { value: 'zh-HK-HiuGaaiNeural', zhLabel: '晓佳 · 粤语女声', enLabel: 'HiuGaai · Cantonese female' },
  { value: 'zh-HK-HiuMaanNeural', zhLabel: '晓曼 · 粤语女声', enLabel: 'HiuMaan · Cantonese female' },
  { value: 'zh-HK-WanLungNeural', zhLabel: '云龙 · 粤语男声', enLabel: 'WanLung · Cantonese male' },
  { value: 'zh-TW-HsiaoChenNeural', zhLabel: '晓臻 · 台湾国语女声', enLabel: 'HsiaoChen · Taiwan female' },
  { value: 'zh-TW-HsiaoYuNeural', zhLabel: '晓雨 · 台湾国语女声', enLabel: 'HsiaoYu · Taiwan female' },
  { value: 'zh-TW-YunJheNeural', zhLabel: '云哲 · 台湾国语男声', enLabel: 'YunJhe · Taiwan male' },
  { value: 'en-US-ChristopherNeural', zhLabel: 'Christopher · 美式英语男声 · 沉稳权威', enLabel: 'Christopher · US male · Authoritative' },
  { value: 'en-US-JennyNeural', zhLabel: 'Jenny · 美式英语女声 · 亲切自然', enLabel: 'Jenny · US female · Friendly' },
  { value: 'en-US-AriaNeural', zhLabel: 'Aria · 美式英语女声 · 自信清晰', enLabel: 'Aria · US female · Confident' },
  { value: 'en-US-GuyNeural', zhLabel: 'Guy · 美式英语男声 · 富有激情', enLabel: 'Guy · US male · Passionate' },
  { value: 'ja-JP-NanamiNeural', zhLabel: 'Nanami · 日语女声', enLabel: 'Nanami · Japanese female' },
  { value: 'ja-JP-KeitaNeural', zhLabel: 'Keita · 日语男声', enLabel: 'Keita · Japanese male' },
  { value: 'ko-KR-SunHiNeural', zhLabel: 'SunHi · 韩语女声', enLabel: 'SunHi · Korean female' },
];

export function isEdgeTtsVoice(voice: string): boolean {
  return EDGE_TTS_VOICES.some((item) => item.value === voice);
}

export function normalizeTtsVoice(engine: TtsEngine, voice: string | undefined): string {
  if (engine === 'edge') {
    return voice && isEdgeTtsVoice(voice) ? voice : DEFAULT_EDGE_TTS_VOICE;
  }
  return voice && !isEdgeTtsVoice(voice) ? voice : DEFAULT_OPENAI_TTS_VOICE;
}
