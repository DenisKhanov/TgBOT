package constant

const (
	EMOJI_BICEPS                         = "\U0001F4AA"           //💪
	EMOJI_GRINNING_FACE                  = "\U0001F600"           //😀
	EMOJI_SMILING_FACE_WITH_HEART_EYES   = "\U0001F60D"           //😍
	EMOJI_FACE_WITH_TEARS_OF_JOY         = "\U0001F602"           //😂
	EMOJI_SMILING_FACE_WITH_SMILING_EYES = "\U0001F60A"           //😊
	EMOJI_RED_HEART                      = "\U00002764\U0000FE0F" //❤️
	EMOJI_THUMBS_UP                      = "\U0001F44D"           //👍
	EMOJI_FIRE                           = "\U0001F525"           //🔥
	EMOJI_CLAPPING_HANDS                 = "\U0001F44F"           //👏
	EMOJI_FOLDED_HANDS                   = "\U0001F64F"           //🙏
	EMOJI_WINKING_FACE                   = "\U0001F609"           //😉
	EMOJI_SMILING_FACE_WITH_SUNGLASSES   = "\U0001F60E"           //😎
	EMOJI_CHECK_MARK                     = "\U00002714\U0000FE0F" //✔️
	EMOJI_EYES                           = "\U0001F440"           //👀
	EMOJI_CRYING_FACE                    = "\U0001F622"           //😢
	EMOJI_FACE_SCREAMING_IN_FEAR         = "\U0001F631"           //😱
	EMOJI_BUTTON_START                   = "\U000025B6  "         // ▶
	EMOJI_BUTTON_END                     = "  \U000025C0"         // ◀
	EMOJI_BUTTON_UP                      = "\U0001F199"           //🆙

	BUTTON_TEXT_PRINT_INTRO                = EMOJI_BUTTON_START + "Расскажи о себе" + EMOJI_BUTTON_END
	BUTTON_TEXT_SKIP_INTRO                 = EMOJI_BUTTON_START + "Пропусти вступление" + EMOJI_BUTTON_END
	BUTTON_TEXT_WHAT_TO_DO                 = EMOJI_BUTTON_START + "Чем мне заняться?" + EMOJI_BUTTON_END
	BUTTON_TEXT_TRANSLATE                  = EMOJI_BUTTON_START + "Переведи текст" + EMOJI_BUTTON_END
	BUTTON_TEXT_YANDEX_TURN_ON_NIGHT_LIGHT = EMOJI_BUTTON_START + "Включи/Выключи ночник" + EMOJI_BUTTON_END
	BUTTON_TEXT_YANDEX_TURN_ON_SPEAKER     = EMOJI_BUTTON_START + "Включи/Выключи колонки" + EMOJI_BUTTON_END
	BUTTON_TEXT_YANDEX_DDIALOGS            = EMOJI_BUTTON_START + "Меню Yandex диалогов" + EMOJI_BUTTON_END
	BUTTON_TEXT_YANDEX_LOGIN               = EMOJI_BUTTON_START + "Получить токен" + EMOJI_BUTTON_END
	BUTTON_TEXT_YANDEX_SEND_CODE           = EMOJI_BUTTON_START + "Пройти аутентификацию" + EMOJI_BUTTON_END
	BUTTON_TEXT_YANDEX_GET_HOME_INFO       = EMOJI_BUTTON_START + "Показать информацию SmartHome" + EMOJI_BUTTON_END

	BUTTON_TEXT_PRINT_MENU = EMOJI_BUTTON_START + "Покажи главное меню" + EMOJI_BUTTON_END

	BUTTON_CODE_PRINT_INTRO                = "print_intro"
	BUTTON_CODE_SKIP_INTRO                 = "skip_intro"
	BUTTON_CODE_WHAT_TO_DO                 = "what_should_i_do"
	BUTTON_CODE_TRANSLATE                  = "text_translate"
	BUTTON_CODE_YANDEX_TURN_ON_NIGHT_LIGHT = "turn_on_off_night_light"
	BUTTON_CODE_YANDEX_TURN_ON_SPEAKER     = "turn_on_off_speaker"
	BUTTON_CODE_YANDEX_DDIALOGS            = "yandex_dialogs"
	BUTTON_CODE_YANDEX_LOGIN               = "yandex_login"
	BUTTON_CODE_YANDEX_SEND_CODE           = "https://oauth.yandex.ru/authorize?response_type=code&client_id=f78d9fab1f2b49ca9c729ec0c72964a8&redirect_uri=https://localhost:8080/callback&state="
	BUTTON_CODE_YANDEX_GET_HOME_INFO       = "yandex_home_info"
	BUTTON_CODE_PRINT_MENU                 = "print_menu"
)
