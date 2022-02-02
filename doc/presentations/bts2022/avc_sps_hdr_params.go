func avcHdrParameters(d *decode.D) {
	cpbCnt := d.FieldUFn("cpb_cnt", uEV, scalar.UAdd(1))
	d.FieldU4("bit_rate_scale")
	d.FieldU4("cpb_size_scale")
	d.FieldArray("sched_sels", func(d *decode.D) {
		for i := uint64(0); i < cpbCnt; i++ {
			d.FieldStruct("sched_sel", func(d *decode.D) {
				d.FieldUFn("bit_rate_value", uEV, scalar.UAdd(1))
				d.FieldUFn("cpb_size_value", uEV, scalar.UAdd(1))
				d.FieldBool("cbr_flag")
			})
		}
	})
	d.FieldU5("initial_cpb_removal_delay_length", scalar.UAdd(1))
	d.FieldU5("cpb_removal_delay_length", scalar.UAdd(1))
	d.FieldU5("dpb_output_delay_length", scalar.UAdd(1))
	d.FieldU5("time_offset_length")
}