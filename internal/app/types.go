package app

type PhraseWithTranslation struct {
	Phrase, Translation string
}

func (pwt *PhraseWithTranslation) Invert() {
	pwt.Phrase, pwt.Translation = pwt.Translation, pwt.Phrase
}
