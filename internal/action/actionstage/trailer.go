package actionstage

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Galdoba/lazyam/internal/appmodule/config"
	"github.com/Galdoba/lazyam/pkg/scriptkit"
)

const (
	amediaTrailerTail = "_a_teka.mp4"
)

func testlist() []string {
	return []string{
		"absolyutnoe_zlo_ondscan_treyler_a_teka.mp4",
		"Adskiy_uchitel_Nube_Treyler_A-teka.mp4",
		"amerikanskaya_rzhavchina_2s_treyler_a_teka.mp4",
		"Art_detektivy_Treyler_A-teka.mp4",
		"Barri_4s_treyler_a_teka.mp4",
		"Bistro_La_Favorita_2c_Treyler_A-teka.mp4",
		"bosh_1_7s_treyler_a_teka.mp4",
		"Byvshaya_zhena_2s_Treyler_A-teka.mp4",
		"chelovek_kotoryy_upal_na_zemlyu_treyler_a_teka.mp4",
		"chuzhoe_telo_treyler_a_teka.mp4",
		"Dandadan_1s_Treyler_A-teka.mp4",
		"Dandadan_2s_Treyler_A-teka.mp4",
		"Doktor_Doktor_Treyler_A-teka.mp4",
		"ee_istoriya_treyler_a_teka.mp4",
		"Eti_hrabrye_devochki_2s_Treyler_A-teka.mp4",
		"Fioletovyy_kak_more_Treyler_A-teka.mp4",
		"Gostya_Treyler_A-teka.mp4",
		"halo_2s_treyler_1_a_teka.mp4",
		"halo_treyler_a_teka.mp4",
		"horoshaya_borba_6s_treyler_a_teka.mp4",
		"Horoshiy_kop_plohoy_kop_Treyler_A-teka.mp4",
		"intervyu_s_vampirom_2s_treyler_a_teka.mp4",
		"intervyu_s_vampirom_treyler_a_teka.mp4",
		"Irlandskaya_krov_Treyler_A-teka.mp4",
		"Ischeznuvshie_2s_Treyler_A-teka.mp4",
		"Kassetomaniya_Treyler_A-teka.mp4",
		"kevin_mozhet_poyti_k_chertu_treyler_a_teka.mp4",
		"Kompyutershiki_Treyler_A-teka.mp4",
		"lozha_49_treyler_a_teka.mp4",
		"malenkaya_barabanshica_treyler_a_teka.mp4",
		"melanholiya_treyler_a_teka.mp4",
		"mese_speyd_treyler_a_teka.mp4",
		"milliardy_7s_treyler_a_teka.mp4",
		"molodoy_papa_treyler_a_teka.mp4",
		"Na_podeme_Treyler_A-teka.mp4",
		"Nasledniki_4s_treyler_2_a_teka.mp4",
		"nastoyashiy_detektiv_4s_treyler_a_teka.mp4",
		"navsegda_treyler_a_teka.mp4",
		"Perri_meyson_2s_treyler_a_teka.mp4",
		"Pod_podozreniem_Treyler_A-teka.mp4",
		"pod_vinogradnoy_lozoy_treyler_a_teka.mp4",
		"poymay_mne_ubiycu_treyler_a_teka.mp4",
		"pozolochennyy_vek_2s_treyler_2_a_teka.mp4",
		"Pravednye_dzhemstouny_treyler_3s_a_teka.mp4",
		"prizraki_2s_treyler_a_teka.mp4",
		"prizraki_4s_treyler_a_teka.mp4",
		"Rassledovaniya_Merdoka_Treyler_A-teka.mp4",
		"Shababniki_3s_Treyler_A-teka.mp4",
		"Shababniki_treyler_a_teka.mp4",
		"Sherlok_i_doch_Treyler_A-teka.mp4",
		"Shershni_Treyler_2_3s_A-teka.mp4",
		"shershni_treyler_2s_a_teka.mp4",
		"Skvoz_sneg_Treyler_A-teka.mp4",
		"slushateli_treyler_a_teka.mp4",
		"Son_vo_sne_Treyler_A-teka.mp4",
		"Strana_Rozhdestva_Treyler_A-teka.mp4",
		"temnoe_ditya_otgoloski_treyler_a_teka.mp4",
		"temnye_vetra_treyler_a_teka.mp4",
		"Temnyy_demon_Treyler_A-teka.mp4",
		"terror_treyler_a_teka.mp4",
		"ubezhishe_treyler_a_teka.mp4",
		"uoker_4s_treyler_a_teka.mp4",
		"uroki_kitayskogo_treyler_a_teka.mp4",
		"vasha_chest_2s_treyler_a_teka.mp4",
		"Viktoriya_Treyler_A-teka.mp4",
		"Voin_3s_treyler_a_teka.mp4",
		"volk_treyler_a_teka.mp4",
		"vygodnaya_afera_treyler_a_teka.mp4",
		"Yarmarka_Tsheslaviya_Treyler_A-teka.mp4",
		"zlo_4s_treyler_a_teka.mp4",
		"Zona_interesov_Treyler_A-teka.mp4",
		"zorro_treyler_a_teka.mp4",
	}
}

func WriteScript(cfg *config.Config, path string) error {
	if !IsAmediaTrailer(path) {
		return nil
	}
	dir := filepath.Dir(path)
	source := filepath.Base(path)
	outbase := trailerName(path) + trailerSeason(path)
	script := scriptkit.New(filepath.ToSlash(filepath.Join(dir, outbase+".sh")), scriptkit.WithTemplate(scriptkit.AmediaTrailer),
		scriptkit.WithArgs(
			scriptkit.ScriptArg("source", source),
			scriptkit.ScriptArg("outbase", outbase),
		),
	)
	return script.CreateScriptFile()
}

func IsAmediaTrailer(path string) bool {
	file := filepath.Base(path)
	lowName := strings.ToLower(file)
	lowName = strings.ReplaceAll(lowName, "-", "_")
	if !strings.Contains(lowName, amediaTrailerTail) {
		return false
	}
	return true
}

func Outname(path string) string {
	file := strings.TrimSuffix(path, amediaTrailerTail)
	file = filepath.Base(file)
	lowName := strings.ToLower(file)
	lowName = strings.ReplaceAll(lowName, "-", "_")
	out := trailerName(path) + trailerSeason(lowName)

	return out
}

func trailerSeason(base string) string {
	for i := 1; i < 100; i++ {
		seasonMark := fmt.Sprintf("_%vs", i)
		if strings.Contains(base, seasonMark) {
			return fmt.Sprintf("_s%v", num(i))
		}
	}
	return ""
}

func num(n int) string {
	s := fmt.Sprintf("%v", n)
	for len(s) < 2 {
		s = "0" + s
	}
	return s
}

func trailerName(path string) string {
	file := filepath.Base(path)
	lowName := strings.ToLower(file)
	lowName = strings.ReplaceAll(lowName, "-", "_")
	for i := 1; i < 100; i++ {
		if strings.Contains(lowName, fmt.Sprintf("_%vs_", i)) {
			lowName = strings.ReplaceAll(lowName, fmt.Sprintf("_%vs_", i), "_")
		}
	}
	lowName = strings.ReplaceAll(lowName, "_treyler", "")
	lowName = strings.ReplaceAll(lowName, ".mp4", "")
	lowName = strings.ReplaceAll(lowName, "_a_teka", "")
	for i := 1; i < 100; i++ {
		if strings.Contains(lowName, fmt.Sprintf("_%vs_", i)) {
			lowName = strings.ReplaceAll(lowName, fmt.Sprintf("_%vs", i), "")
		}
	}
	lowName = strings.ReplaceAll(lowName, "__", "_")
	letters := strings.Split(lowName, "")
	letters[0] = strings.ToUpper(letters[0])
	return strings.Join(letters, "")
}
