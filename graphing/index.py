import re
from pathlib import Path

import pandas as pd
import matplotlib.pyplot as plt


RAW_TEXT = r"""
Password Ace Worker 1
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=32.729µs avg=32.729µs min=32.729µs max=32.729µs
- job dispatch/registration overhead: count=9 total=133.293429ms avg=14.810381ms min=11.07984ms max=17.954473ms
- work assignment overhead: count=9 total=48.242597ms avg=5.360288ms min=546.999µs max=8.47388ms | per-unit=5360.29ns (units=9000)
- worker cracking time (compute/search): count=9 total=11.398773784s avg=1.26653042s min=736.415411ms max=1.382311679s
- result return latency (worker -> controller): count=1 total=317.582µs avg=317.582µs min=317.582µs max=317.582µs
- checkpoint overhead observations: count=85 total=493.275996ms avg=5.803247ms min=4.97496ms max=17.706318ms
- total end-to-end runtime: count=1 total=15.984453238s avg=15.984453238s min=15.984453238s max=15.984453238s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 48.275326ms
  networking overhead: 133.611011ms
  checkpoint overhead: 493.275996ms (moderate impact, 3.09% of end-to-end)
  combined overhead:   675.162333ms




Password Ace Worker 2
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=547.973µs avg=547.973µs min=547.973µs max=547.973µs
- job dispatch/registration overhead: count=9 total=145.981998ms avg=16.220222ms min=9.854092ms max=30.45606ms
- work assignment overhead: count=8 total=35.323028ms avg=4.415378ms min=542.167µs max=5.970135ms | per-unit=4415.38ns (units=8000)
- worker cracking time (compute/search): count=8 total=9.869248023s avg=1.233656002s min=759.347767ms max=1.319021158s
- result return latency (worker -> controller): count=1 total=333.367µs avg=333.367µs min=333.367µs max=333.367µs
- checkpoint overhead observations: count=82 total=499.801391ms avg=6.095138ms min=5.044744ms max=18.005221ms
- total end-to-end runtime: count=1 total=6.340247632s avg=6.340247632s min=6.340247632s max=6.340247632s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 35.871001ms
  networking overhead: 146.315365ms
  checkpoint overhead: 499.801391ms (high impact, 7.88% of end-to-end)
  combined overhead:   681.987757ms




Password Ace Worker 3
  ===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=985.459µs avg=985.459µs min=985.459µs max=985.459µs
- job dispatch/registration overhead: count=10 total=150.511133ms avg=15.051113ms min=9.952493ms max=20.37155ms
- work assignment overhead: count=8 total=33.677799ms avg=4.209724ms min=597.678µs max=7.777268ms | per-unit=4209.72ns (units=8000)
- worker cracking time (compute/search): count=8 total=9.911205317s avg=1.238900664s min=734.107991ms max=1.335132216s
- result return latency (worker -> controller): count=1 total=285.569µs avg=285.569µs min=285.569µs max=285.569µs
- checkpoint overhead observations: count=84 total=529.268695ms avg=6.300817ms min=4.954392ms max=18.488848ms
- total end-to-end runtime: count=1 total=13.645110187s avg=13.645110187s min=13.645110187s max=13.645110187s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 34.663258ms
  networking overhead: 150.796702ms
  checkpoint overhead: 529.268695ms (moderate impact, 3.88% of end-to-end)
  combined overhead:   714.728655ms





Password Ace Worker 5
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=2.32254ms avg=2.32254ms min=2.32254ms max=2.32254ms
- job dispatch/registration overhead: count=12 total=165.739977ms avg=13.811664ms min=9.608395ms max=18.171353ms
- work assignment overhead: count=8 total=26.87676ms avg=3.359595ms min=318.662µs max=8.417674ms | per-unit=3359.59ns (units=8000)
- worker cracking time (compute/search): count=8 total=10.118066571s avg=1.264758321s min=749.790687ms max=1.363170409s
- result return latency (worker -> controller): count=1 total=345.193µs avg=345.193µs min=345.193µs max=345.193µs
- checkpoint overhead observations: count=89 total=537.670143ms avg=6.041237ms min=5.05604ms max=16.212445ms
- total end-to-end runtime: count=1 total=5.237835341s avg=5.237835341s min=5.237835341s max=5.237835341s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 29.1993ms
  networking overhead: 166.08517ms
  checkpoint overhead: 537.670143ms (high impact, 10.27% of end-to-end)
  combined overhead:   732.954613ms





Password Bad Worker 1
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=598.565µs avg=598.565µs min=598.565µs max=598.565µs
- job dispatch/registration overhead: count=15 total=214.532446ms avg=14.302163ms min=9.293679ms max=21.637092ms
- work assignment overhead: count=15 total=81.795756ms avg=5.45305ms min=685.078µs max=9.988845ms | per-unit=5453.05ns (units=15000)
- worker cracking time (compute/search): count=15 total=19.347603472s avg=1.289840231s min=837.69743ms max=1.368393691s
- result return latency (worker -> controller): count=1 total=360.437µs avg=360.437µs min=360.437µs max=360.437µs
- checkpoint overhead observations: count=146 total=861.781524ms avg=5.902613ms min=2.841464ms max=22.586794ms
- total end-to-end runtime: count=1 total=22.122574053s avg=22.122574053s min=22.122574053s max=22.122574053s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 82.394321ms
  networking overhead: 214.892883ms
  checkpoint overhead: 861.781524ms (moderate impact, 3.90% of end-to-end)
  combined overhead:   1.159068728s




Password Bad Worker 2
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=511.905µs avg=511.905µs min=511.905µs max=511.905µs
- job dispatch/registration overhead: count=16 total=246.146835ms avg=15.384177ms min=10.790674ms max=23.517101ms
- work assignment overhead: count=15 total=78.68278ms avg=5.245518ms min=433.269µs max=8.005987ms | per-unit=5245.52ns (units=15000)
- worker cracking time (compute/search): count=15 total=19.509856707s avg=1.300657113s min=887.614591ms max=1.383137393s
- result return latency (worker -> controller): count=1 total=430.681µs avg=430.681µs min=430.681µs max=430.681µs
- checkpoint overhead observations: count=151 total=942.955194ms avg=6.244736ms min=5.009305ms max=24.770662ms
- total end-to-end runtime: count=1 total=13.742855529s avg=13.742855529s min=13.742855529s max=13.742855529s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 79.194685ms
  networking overhead: 246.577516ms
  checkpoint overhead: 942.955194ms (high impact, 6.86% of end-to-end)
  combined overhead:   1.268727395s




Password Bad Worker 3
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=2.967175ms avg=2.967175ms min=2.967175ms max=2.967175ms
- job dispatch/registration overhead: count=15 total=230.529303ms avg=15.36862ms min=10.794735ms max=24.210989ms
- work assignment overhead: count=13 total=68.352382ms avg=5.257875ms min=603.233µs max=13.970061ms | per-unit=5257.88ns (units=13000)
- worker cracking time (compute/search): count=13 total=16.658127372s avg=1.281394413s min=885.923726ms max=1.383429594s
- result return latency (worker -> controller): count=1 total=364.935µs avg=364.935µs min=364.935µs max=364.935µs
- checkpoint overhead observations: count=141 total=886.76814ms avg=6.289135ms min=4.947735ms max=20.443034ms
- total end-to-end runtime: count=1 total=18.113060535s avg=18.113060535s min=18.113060535s max=18.113060535s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 71.319557ms
  networking overhead: 230.894238ms
  checkpoint overhead: 886.76814ms (moderate impact, 4.90% of end-to-end)
  combined overhead:   1.188981935s





Password Bad Worker 5
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=601.609µs avg=601.609µs min=601.609µs max=601.609µs
- job dispatch/registration overhead: count=17 total=264.206143ms avg=15.541537ms min=10.25756ms max=21.078567ms
- work assignment overhead: count=13 total=54.897584ms avg=4.222891ms min=309.33µs max=8.903458ms | per-unit=4222.89ns (units=13000)
- worker cracking time (compute/search): count=13 total=16.639945306s avg=1.279995792s min=845.580834ms max=1.362131229s
- result return latency (worker -> controller): count=1 total=322.377µs avg=322.377µs min=322.377µs max=322.377µs
- checkpoint overhead observations: count=145 total=946.18744ms avg=6.52543ms min=4.884339ms max=26.465023ms
- total end-to-end runtime: count=1 total=5.927112001s avg=5.927112001s min=5.927112001s max=5.927112001s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 55.499193ms
  networking overhead: 264.52852ms
  checkpoint overhead: 946.18744ms (high impact, 15.96% of end-to-end)
  combined overhead:   1.266215153s





Password Cab Worker 1
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=616.361µs avg=616.361µs min=616.361µs max=616.361µs
- job dispatch/registration overhead: count=21 total=366.700523ms avg=17.461929ms min=13.405513ms max=60.881557ms
- work assignment overhead: count=21 total=160.549854ms avg=7.645231ms min=447.333µs max=54.889822ms | per-unit=7645.23ns (units=21000)
- worker cracking time (compute/search): count=21 total=27.368996964s avg=1.303285569s min=1.166429589s max=1.360529507s
- result return latency (worker -> controller): count=1 total=343.472µs avg=343.472µs min=343.472µs max=343.472µs
- checkpoint overhead observations: count=208 total=1.285117262s avg=6.178448ms min=4.836577ms max=66.857124ms
- total end-to-end runtime: count=1 total=29.739544795s avg=29.739544795s min=29.739544795s max=29.739544795s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 161.166215ms
  networking overhead: 367.043995ms
  checkpoint overhead: 1.285117262s (moderate impact, 4.32% of end-to-end)
  combined overhead:   1.813327472s




Password Cab Worker 2
- controller-side parsing time: count=1 total=2.393922ms avg=2.393922ms min=2.393922ms max=2.393922ms
- job dispatch/registration overhead: count=22 total=345.615585ms avg=15.709799ms min=10.684107ms max=33.613582ms
- work assignment overhead: count=21 total=111.23026ms avg=5.296679ms min=448.187µs max=8.907313ms | per-unit=5296.68ns (units=21000)
- worker cracking time (compute/search): count=21 total=27.292197581s avg=1.299628456s min=1.137404649s max=1.345951933s
- result return latency (worker -> controller): count=1 total=306.741µs avg=306.741µs min=306.741µs max=306.741µs
- checkpoint overhead observations: count=215 total=1.341818618s avg=6.241016ms min=4.845168ms max=34.203622ms
- total end-to-end runtime: count=1 total=16.064055635s avg=16.064055635s min=16.064055635s max=16.064055635s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 113.624182ms
  networking overhead: 345.922326ms
  checkpoint overhead: 1.341818618s (high impact, 8.35% of end-to-end)
  combined overhead:   1.801365126s




Password Cab Worker 3
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=572.76µs avg=572.76µs min=572.76µs max=572.76µs
- job dispatch/registration overhead: count=23 total=340.985932ms avg=14.825475ms min=10.0644ms max=17.646453ms
- work assignment overhead: count=21 total=102.048954ms avg=4.859474ms min=399.258µs max=6.271847ms | per-unit=4859.47ns (units=21000)
- worker cracking time (compute/search): count=21 total=27.402232377s avg=1.304868208s min=1.147081996s max=1.369105088s
- result return latency (worker -> controller): count=1 total=407.608µs avg=407.608µs min=407.608µs max=407.608µs
- checkpoint overhead observations: count=213 total=1.330036987s avg=6.244305ms min=4.971795ms max=19.419495ms
- total end-to-end runtime: count=1 total=12.037563992s avg=12.037563992s min=12.037563992s max=12.037563992s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 102.621714ms
  networking overhead: 341.39354ms
  checkpoint overhead: 1.330036987s (high impact, 11.05% of end-to-end)
  combined overhead:   1.774052241s



Password Cab Worker 5
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=64.228µs avg=64.228µs min=64.228µs max=64.228µs
- job dispatch/registration overhead: count=25 total=387.961768ms avg=15.51847ms min=9.918475ms max=22.889037ms
- work assignment overhead: count=21 total=101.730757ms avg=4.844321ms min=421.843µs max=9.841224ms | per-unit=4844.32ns (units=21000)
- worker cracking time (compute/search): count=21 total=27.867305175s avg=1.327014532s min=1.187360396s max=1.413773757s
- result return latency (worker -> controller): count=1 total=346.456µs avg=346.456µs min=346.456µs max=346.456µs
- checkpoint overhead observations: count=221 total=1.403875277s avg=6.352376ms min=4.926331ms max=22.233524ms
- total end-to-end runtime: count=1 total=8.718654395s avg=8.718654395s min=8.718654395s max=8.718654395s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 101.794985ms
  networking overhead: 388.308224ms
  checkpoint overhead: 1.403875277s (high impact, 16.10% of end-to-end)
  combined overhead:   1.893978486s




Password Dad Worker 1
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=554.102µs avg=554.102µs min=554.102µs max=554.102µs
- job dispatch/registration overhead: count=28 total=429.466219ms avg=15.338079ms min=11.672591ms max=18.669758ms
- work assignment overhead: count=28 total=153.887353ms avg=5.495976ms min=495.235µs max=8.216611ms | per-unit=5495.98ns (units=28000)
- worker cracking time (compute/search): count=28 total=35.269688018s avg=1.259631714s min=163.334022ms max=1.395953669s
- result return latency (worker -> controller): count=1 total=334.952µs avg=334.952µs min=334.952µs max=334.952µs
- checkpoint overhead observations: count=271 total=1.735327857s avg=6.403423ms min=5.030963ms max=26.688831ms
- total end-to-end runtime: count=1 total=37.35632461s avg=37.35632461s min=37.35632461s max=37.35632461s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 154.441455ms
  networking overhead: 429.801171ms
  checkpoint overhead: 1.735327857s (moderate impact, 4.65% of end-to-end)
  combined overhead:   2.319570483s




Password Dad Worker 2
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=2.393118ms avg=2.393118ms min=2.393118ms max=2.393118ms
- job dispatch/registration overhead: count=28 total=416.115963ms avg=14.861284ms min=10.678874ms max=18.091372ms
- work assignment overhead: count=27 total=142.548273ms avg=5.279565ms min=304.932µs max=7.130386ms | per-unit=5279.57ns (units=27000)
- worker cracking time (compute/search): count=27 total=34.309117095s avg=1.27070804s min=165.172894ms max=1.348590236s
- result return latency (worker -> controller): count=1 total=406.112µs avg=406.112µs min=406.112µs max=406.112µs
- checkpoint overhead observations: count=270 total=1.664555894s avg=6.165021ms min=4.936946ms max=23.81527ms
- total end-to-end runtime: count=1 total=18.767757168s avg=18.767757168s min=18.767757168s max=18.767757168s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 144.941391ms
  networking overhead: 416.522075ms
  checkpoint overhead: 1.664555894s (high impact, 8.87% of end-to-end)
  combined overhead:   2.22601936s




Password Dad Worker 3
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=631.081µs avg=631.081µs min=631.081µs max=631.081µs
- job dispatch/registration overhead: count=30 total=451.9247ms avg=15.064156ms min=10.669615ms max=20.209473ms
- work assignment overhead: count=28 total=148.675429ms avg=5.309836ms min=453.707µs max=7.734798ms | per-unit=5309.84ns (units=28000)
- worker cracking time (compute/search): count=28 total=35.878326965s avg=1.28136882s min=168.0746ms max=1.404559625s
- result return latency (worker -> controller): count=1 total=310.807µs avg=310.807µs min=310.807µs max=310.807µs
- checkpoint overhead observations: count=271 total=1.722302159s avg=6.355358ms min=5.028086ms max=20.354057ms
- total end-to-end runtime: count=1 total=14.061031919s avg=14.061031919s min=14.061031919s max=14.061031919s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 149.30651ms
  networking overhead: 452.235507ms
  checkpoint overhead: 1.722302159s (high impact, 12.25% of end-to-end)
  combined overhead:   2.323844176s




Password Dad Worker 5
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=42.936µs avg=42.936µs min=42.936µs max=42.936µs
- job dispatch/registration overhead: count=28 total=444.798015ms avg=15.885643ms min=10.308168ms max=21.945728ms
- work assignment overhead: count=24 total=122.823556ms avg=5.117648ms min=358.064µs max=11.857446ms | per-unit=5117.65ns (units=24000)
- worker cracking time (compute/search): count=24 total=30.282363065s avg=1.261765127s min=161.532269ms max=1.33214042s
- result return latency (worker -> controller): count=1 total=423.571µs avg=423.571µs min=423.571µs max=423.571µs
- checkpoint overhead observations: count=249 total=1.532286255s avg=6.15376ms min=4.973883ms max=21.594235ms
- total end-to-end runtime: count=1 total=9.418923719s avg=9.418923719s min=9.418923719s max=9.418923719s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 122.866492ms
  networking overhead: 445.221586ms
  checkpoint overhead: 1.532286255s (high impact, 16.27% of end-to-end)
  combined overhead:   2.100374333s





Password Ear Worker 1
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=495.019µs avg=495.019µs min=495.019µs max=495.019µs
- job dispatch/registration overhead: count=34 total=549.788668ms avg=16.170254ms min=14.294737ms max=25.770127ms
- work assignment overhead: count=34 total=187.402131ms avg=5.511827ms min=341.085µs max=7.562308ms | per-unit=5511.83ns (units=34000)
- worker cracking time (compute/search): count=34 total=43.353517731s avg=1.275103462s min=496.926035ms max=1.340598088s
- result return latency (worker -> controller): count=1 total=361.48µs avg=361.48µs min=361.48µs max=361.48µs
- checkpoint overhead observations: count=333 total=2.03962499s avg=6.124999ms min=4.861944ms max=26.345092ms
- total end-to-end runtime: count=1 total=43.990097333s avg=43.990097333s min=43.990097333s max=43.990097333s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 187.89715ms
  networking overhead: 550.150148ms
  checkpoint overhead: 2.03962499s (moderate impact, 4.64% of end-to-end)
  combined overhead:   2.777672288s




Password Ear Worker 2
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=558.155µs avg=558.155µs min=558.155µs max=558.155µs
- job dispatch/registration overhead: count=34 total=555.42052ms avg=16.335897ms min=9.98262ms max=59.721239ms
- work assignment overhead: count=33 total=206.150565ms avg=6.246986ms min=698.915µs max=33.492776ms | per-unit=6246.99ns (units=33000)
- worker cracking time (compute/search): count=33 total=42.77220796s avg=1.296127513s min=496.359518ms max=1.454176549s
- result return latency (worker -> controller): count=1 total=321.904µs avg=321.904µs min=321.904µs max=321.904µs
- checkpoint overhead observations: count=332 total=2.169429306s avg=6.534425ms min=4.939216ms max=46.779899ms
- total end-to-end runtime: count=1 total=23.699949608s avg=23.699949608s min=23.699949608s max=23.699949608s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 206.70872ms
  networking overhead: 555.742424ms
  checkpoint overhead: 2.169429306s (high impact, 9.15% of end-to-end)
  combined overhead:   2.93188045s




Password Ear Worker 3
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=575.37µs avg=575.37µs min=575.37µs max=575.37µs
- job dispatch/registration overhead: count=36 total=537.190711ms avg=14.921964ms min=9.93148ms max=20.908478ms
- work assignment overhead: count=34 total=181.358218ms avg=5.334065ms min=324.833µs max=11.026071ms | per-unit=5334.07ns (units=34000)
- worker cracking time (compute/search): count=34 total=44.008027102s avg=1.294353738s min=500.587501ms max=1.382185416s
- result return latency (worker -> controller): count=1 total=530.841µs avg=530.841µs min=530.841µs max=530.841µs
- checkpoint overhead observations: count=334 total=1.973302173s avg=5.90809ms min=4.964412ms max=20.301883ms
- total end-to-end runtime: count=1 total=16.392842194s avg=16.392842194s min=16.392842194s max=16.392842194s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 181.933588ms
  networking overhead: 537.721552ms
  checkpoint overhead: 1.973302173s (high impact, 12.04% of end-to-end)
  combined overhead:   2.692957313s




  Password Ear Worker 5
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=668.676µs avg=668.676µs min=668.676µs max=668.676µs
- job dispatch/registration overhead: count=36 total=564.804312ms avg=15.689008ms min=10.180728ms max=23.91123ms
- work assignment overhead: count=33 total=176.506411ms avg=5.348679ms min=352.545µs max=12.384866ms | per-unit=5348.68ns (units=33000)
- worker cracking time (compute/search): count=33 total=42.984723304s avg=1.302567372s min=495.284614ms max=1.381738843s
- result return latency (worker -> controller): count=1 total=364.479µs avg=364.479µs min=364.479µs max=364.479µs
- checkpoint overhead observations: count=332 total=2.047913669s avg=6.168414ms min=4.808256ms max=30.891653ms
- total end-to-end runtime: count=1 total=13.25300485s avg=13.25300485s min=13.25300485s max=13.25300485s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 177.175087ms
  networking overhead: 565.168791ms
  checkpoint overhead: 2.047913669s (high impact, 15.45% of end-to-end)
  combined overhead:   2.790257547s

"""

def duration_to_ms(value: str) -> float:
    value = value.strip().replace("μ", "µ")
    m = re.match(r"([0-9]*\.?[0-9]+)\s*(ns|µs|us|ms|s)$", value)
    if not m:
        raise ValueError(f"Unrecognized duration: {value}")

    num = float(m.group(1))
    unit = m.group(2)

    if unit == "ns":
        return num / 1_000_000.0
    if unit in ("µs", "us"):
        return num / 1000.0
    if unit == "ms":
        return num
    if unit == "s":
        return num * 1000.0

    raise ValueError(f"Unsupported unit: {unit}")


def parse_impact_percent(text: str):
    m = re.search(r"([0-9]+(?:\.[0-9]+)?)%\s+of end-to-end", text)
    return float(m.group(1)) if m else None


def safe_slug(value: str) -> str:
    value = value.strip().lower()
    value = re.sub(r"[^a-z0-9]+", "_", value)
    return value.strip("_")


def parse_runs(raw_text: str):
    """
    Parse blocks like:
      Password Ace Worker 1
      ...
    and normalize to:
      password_label, number_of_workers, metrics...
    """
    lines = [line.rstrip() for line in raw_text.splitlines()]

    run_header_re = re.compile(r"^Password\s+(.+?)\s+Worker\s+(\d+)\s*$")
    metric_re = re.compile(
        r"^- (.*?): count=(\d+)\s+total=([^\s]+)\s+avg=([^\s]+)\s+min=([^\s]+)\s+max=([^\s]+)(?:\s+\|\s+per-unit=([0-9]*\.?[0-9]+)ns\s+\(units=(\d+)\))?$"
    )

    summary_rows = []
    metric_rows = []

    current_label = None
    current_num_workers = None
    current_summary = None

    i = 0
    while i < len(lines):
        line = lines[i].strip()

        header_match = run_header_re.match(line)
        if header_match:
            if current_summary:
                summary_rows.append(current_summary)

            current_label = header_match.group(1).strip()
            current_num_workers = int(header_match.group(2))

            current_summary = {
                "password_label": current_label,
                "hash_algorithm": "bcrypt",
                "number_of_workers": current_num_workers,
                "controller_side_parsing_time_ms": None,
                "job_dispatch_registration_overhead_ms": None,
                "work_assignment_overhead_total_ms": None,
                "work_assignment_overhead_per_unit_ns": None,
                "worker_cracking_time_ms": None,
                "result_return_latency_ms": None,
                "checkpoint_overhead_ms": None,
                "checkpoint_impact_pct": None,
                "total_end_to_end_runtime_ms": None,
                "controller_overhead_ms": None,
                "networking_overhead_ms": None,
                "combined_overhead_ms": None,
                "heartbeat_observation_count": None,
                "heartbeat_avg_interval_metric_ms": None,
                "checkpoint_observation_count": None,
                "checkpoint_avg_ms": None,
                "checkpoint_min_ms": None,
                "checkpoint_max_ms": None,
            }
            i += 1
            continue

        metric_match = metric_re.match(line)
        if metric_match and current_label is not None:
            metric_name = metric_match.group(1).strip()
            count = int(metric_match.group(2))
            total_ms = duration_to_ms(metric_match.group(3))
            avg_ms = duration_to_ms(metric_match.group(4))
            min_ms = duration_to_ms(metric_match.group(5))
            max_ms = duration_to_ms(metric_match.group(6))
            per_unit_ns = (
                float(metric_match.group(7)) if metric_match.group(7) else None
            )
            units = int(metric_match.group(8)) if metric_match.group(8) else None

            metric_rows.append(
                {
                    "password_label": current_label,
                    "hash_algorithm": "bcrypt",
                    "number_of_workers": current_num_workers,
                    "metric": metric_name,
                    "count": count,
                    "total_ms": total_ms,
                    "avg_ms": avg_ms,
                    "min_ms": min_ms,
                    "max_ms": max_ms,
                    "per_unit_ns": per_unit_ns,
                    "units": units,
                }
            )

            if metric_name == "controller-side parsing time":
                current_summary["controller_side_parsing_time_ms"] = total_ms
            elif metric_name == "job dispatch/registration overhead":
                current_summary["job_dispatch_registration_overhead_ms"] = total_ms
            elif metric_name == "work assignment overhead":
                current_summary["work_assignment_overhead_total_ms"] = total_ms
                current_summary["work_assignment_overhead_per_unit_ns"] = per_unit_ns
            elif metric_name == "worker cracking time (compute/search)":
                current_summary["worker_cracking_time_ms"] = total_ms
            elif metric_name == "result return latency (worker -> controller)":
                current_summary["result_return_latency_ms"] = total_ms
            elif metric_name == "checkpoint overhead observations":
                current_summary["checkpoint_overhead_ms"] = total_ms
                current_summary["checkpoint_observation_count"] = count
                current_summary["checkpoint_avg_ms"] = avg_ms
                current_summary["checkpoint_min_ms"] = min_ms
                current_summary["checkpoint_max_ms"] = max_ms
            elif metric_name == "total end-to-end runtime":
                current_summary["total_end_to_end_runtime_ms"] = total_ms

            i += 1
            continue

        if line.startswith("controller overhead:") and current_summary is not None:
            current_summary["controller_overhead_ms"] = duration_to_ms(
                line.split(":", 1)[1].strip()
            )
            i += 1
            continue

        if line.startswith("networking overhead:") and current_summary is not None:
            current_summary["networking_overhead_ms"] = duration_to_ms(
                line.split(":", 1)[1].strip()
            )
            i += 1
            continue

        if line.startswith("checkpoint overhead:") and current_summary is not None:
            payload = line.split(":", 1)[1].strip()
            duration_part = payload.split("(", 1)[0].strip()
            current_summary["checkpoint_overhead_ms"] = duration_to_ms(duration_part)
            current_summary["checkpoint_impact_pct"] = parse_impact_percent(payload)
            i += 1
            continue

        if line.startswith("combined overhead:") and current_summary is not None:
            current_summary["combined_overhead_ms"] = duration_to_ms(
                line.split(":", 1)[1].strip()
            )
            i += 1
            continue

        i += 1

    if current_summary:
        summary_rows.append(current_summary)

    summary_df = pd.DataFrame(summary_rows)
    metrics_df = pd.DataFrame(metric_rows)

    if not summary_df.empty:
        summary_df["run_label"] = (
            summary_df["password_label"]
            + " - "
            + summary_df["number_of_workers"].astype(str)
            + " workers"
        )

    if not metrics_df.empty:
        metrics_df["run_label"] = (
            metrics_df["password_label"]
            + " - "
            + metrics_df["number_of_workers"].astype(str)
            + " workers"
        )

    return summary_df, metrics_df


def add_prediction_columns(summary_df: pd.DataFrame) -> pd.DataFrame:
    """
    Predict runtime at 5 workers using 1-3 worker measurements.

    Model:
        T_n = T_1 * (s + (1-s)/n)

    where s is an estimated serial fraction derived from measured
    combined overhead / total runtime across the 1-3 worker runs.
    """
    df = summary_df.copy()
    df["serial_fraction_estimate"] = None
    df["predicted_runtime_5_workers_ms"] = None
    df["measured_runtime_5_workers_ms"] = None
    df["prediction_error_pct"] = None

    for password_label in df["password_label"].dropna().unique():
        subset = df[df["password_label"] == password_label].copy()
        subset = subset.sort_values("number_of_workers")

        one_worker = subset[subset["number_of_workers"] == 1]
        one_to_three = subset[subset["number_of_workers"].isin([1, 2, 3])]

        if one_worker.empty or one_to_three.empty:
            continue

        t1 = float(one_worker.iloc[0]["total_end_to_end_runtime_ms"])

        serial_candidates = []
        for _, row in one_to_three.iterrows():
            total = row.get("total_end_to_end_runtime_ms")
            overhead = row.get("combined_overhead_ms")
            if pd.notna(total) and pd.notna(overhead) and total > 0:
                frac = overhead / total
                frac = max(0.0, min(1.0, frac))
                serial_candidates.append(frac)

        if not serial_candidates:
            continue

        s = sum(serial_candidates) / len(serial_candidates)
        predicted_5 = t1 * (s + (1.0 - s) / 5.0)

        df.loc[df["password_label"] == password_label, "serial_fraction_estimate"] = s
        df.loc[
            df["password_label"] == password_label, "predicted_runtime_5_workers_ms"
        ] = predicted_5

        measured_5 = subset[subset["number_of_workers"] == 5]
        if not measured_5.empty:
            measured_val = float(measured_5.iloc[0]["total_end_to_end_runtime_ms"])
            error_pct = ((measured_val - predicted_5) / predicted_5) * 100.0
            df.loc[
                df["password_label"] == password_label, "measured_runtime_5_workers_ms"
            ] = measured_val
            df.loc[df["password_label"] == password_label, "prediction_error_pct"] = (
                error_pct
            )

    return df


def save_required_measurements_table(summary_df: pd.DataFrame, output_path: Path):
    cols = [
        "password_label",
        "hash_algorithm",
        "number_of_workers",
        "controller_side_parsing_time_ms",
        "job_dispatch_registration_overhead_ms",
        "work_assignment_overhead_total_ms",
        "work_assignment_overhead_per_unit_ns",
        "worker_cracking_time_ms",
        "result_return_latency_ms",
        "checkpoint_overhead_ms",
        "checkpoint_impact_pct",
        "total_end_to_end_runtime_ms",
        "combined_overhead_ms",
    ]

    table_df = summary_df[cols].copy()

    non_numeric_cols = ["password_label", "hash_algorithm"]
    for col in table_df.columns:
        if col not in non_numeric_cols:
            table_df[col] = pd.to_numeric(table_df[col], errors="coerce")

    numeric_cols = table_df.select_dtypes(include=["number"]).columns
    table_df[numeric_cols] = table_df[numeric_cols].round(3)

    fig_height = max(4, 0.38 * len(table_df) + 1.5)
    fig, ax = plt.subplots(figsize=(18, fig_height))
    ax.axis("off")

    tbl = ax.table(
        cellText=table_df.values,
        colLabels=table_df.columns,
        loc="center",
        cellLoc="center",
    )
    tbl.auto_set_font_size(False)
    tbl.set_fontsize(8)
    tbl.scale(1, 1.25)

    plt.tight_layout()
    plt.savefig(output_path, dpi=200, bbox_inches="tight")
    plt.close()

def create_per_experiment_scaling_charts(summary_df: pd.DataFrame, out_dir: Path):
    """
    Separate graphs per experiment label, with x-axis = number_of_workers.
    """
    for label in sorted(summary_df["password_label"].dropna().unique()):
        sub = summary_df[summary_df["password_label"] == label].copy()
        sub = sub.sort_values("number_of_workers")
        slug = safe_slug(label)

        x = sub["number_of_workers"].astype(str)

        plt.figure(figsize=(8, 5))
        plt.bar(x, sub["total_end_to_end_runtime_ms"])
        plt.xlabel("Number of Workers")
        plt.ylabel("End-to-End Runtime (ms)")
        plt.title(f"Scaling Runtime - {label}")
        plt.tight_layout()
        plt.savefig(out_dir / f"runtime_scaling_{slug}.png", dpi=200)
        plt.close()

        plt.figure(figsize=(8, 5))
        idx = range(len(sub))
        c = sub["controller_overhead_ms"].fillna(0)
        n = sub["networking_overhead_ms"].fillna(0)
        ck = sub["checkpoint_overhead_ms"].fillna(0)

        plt.bar(idx, c, label="Controller")
        plt.bar(idx, n, bottom=c, label="Networking")
        plt.bar(idx, ck, bottom=c + n, label="Checkpoint")
        plt.xticks(list(idx), x)
        plt.xlabel("Number of Workers")
        plt.ylabel("Overhead (ms)")
        plt.title(f"Overhead Breakdown - {label}")
        plt.legend()
        plt.tight_layout()
        plt.savefig(out_dir / f"overhead_breakdown_{slug}.png", dpi=200)
        plt.close()

        plt.figure(figsize=(8, 5))
        plt.bar(x, sub["worker_cracking_time_ms"])
        plt.xlabel("Number of Workers")
        plt.ylabel("Worker Cracking Time Total (ms)")
        plt.title(f"Worker Cracking Time - {label}")
        plt.tight_layout()
        plt.savefig(out_dir / f"worker_cracking_time_{slug}.png", dpi=200)
        plt.close()

        plt.figure(figsize=(8, 5))
        plt.bar(x, sub["checkpoint_impact_pct"])
        plt.xlabel("Number of Workers")
        plt.ylabel("Checkpoint Impact (%)")
        plt.title(f"Checkpoint Impact - {label}")
        plt.tight_layout()
        plt.savefig(out_dir / f"checkpoint_impact_{slug}.png", dpi=200)
        plt.close()


def create_prediction_validation_chart(summary_df: pd.DataFrame, out_dir: Path):
    rows = []
    for password_label in sorted(summary_df["password_label"].dropna().unique()):
        subset = summary_df[summary_df["password_label"] == password_label]
        pred_vals = subset["predicted_runtime_5_workers_ms"].dropna().unique()
        meas_vals = subset["measured_runtime_5_workers_ms"].dropna().unique()

        if len(pred_vals) == 0:
            continue

        rows.append(
            {
                "password_label": password_label,
                "predicted_runtime_5_workers_ms": float(pred_vals[0]),
                "measured_runtime_5_workers_ms": (
                    float(meas_vals[0]) if len(meas_vals) else None
                ),
            }
        )

    if not rows:
        return

    df = pd.DataFrame(rows).sort_values("password_label").reset_index(drop=True)

    x = range(len(df))
    width = 0.35

    plt.figure(figsize=(10, 5))
    plt.bar(
        [i - width / 2 for i in x],
        df["predicted_runtime_5_workers_ms"],
        width=width,
        label="Predicted 5 workers",
    )

    measured_vals = df["measured_runtime_5_workers_ms"].fillna(0)
    plt.bar(
        [i + width / 2 for i in x],
        measured_vals,
        width=width,
        label="Measured 5 workers",
    )

    plt.xticks(list(x), df["password_label"], rotation=30, ha="right")
    plt.ylabel("Runtime (ms)")
    plt.title("Predicted vs Measured Runtime at 5 Workers")
    plt.legend()
    plt.tight_layout()
    plt.savefig(out_dir / "prediction_vs_measured_5_workers.png", dpi=200)
    plt.close()


def save_prediction_table(summary_df: pd.DataFrame, output_path: Path):
    pred_df = (
        summary_df[
            [
                "password_label",
                "hash_algorithm",
                "serial_fraction_estimate",
                "predicted_runtime_5_workers_ms",
                "measured_runtime_5_workers_ms",
                "prediction_error_pct",
            ]
        ]
        .drop_duplicates(subset=["password_label"])
        .sort_values("password_label")
        .reset_index(drop=True)
    )

    pred_df = pred_df.dropna(
        how="all",
        subset=[
            "serial_fraction_estimate",
            "predicted_runtime_5_workers_ms",
            "measured_runtime_5_workers_ms",
            "prediction_error_pct",
        ],
    )

    if pred_df.empty:
        return

    for col in pred_df.columns:
        if col not in ("password_label", "hash_algorithm"):
            pred_df[col] = pred_df[col].round(3)

    fig_height = max(3, 0.5 * len(pred_df) + 1.2)
    fig, ax = plt.subplots(figsize=(12, fig_height))
    ax.axis("off")

    tbl = ax.table(
        cellText=pred_df.values,
        colLabels=pred_df.columns,
        loc="center",
        cellLoc="center",
    )
    tbl.auto_set_font_size(False)
    tbl.set_fontsize(8)
    tbl.scale(1, 1.3)

    plt.tight_layout()
    plt.savefig(output_path, dpi=200, bbox_inches="tight")
    plt.close()


def main():
    out_dir = Path("assignment_output_workers_5")
    out_dir.mkdir(exist_ok=True)

    summary_df, metrics_df = parse_runs(RAW_TEXT)

    if summary_df.empty:
        raise RuntimeError("No runs were parsed. Check RAW_TEXT formatting.")

    summary_df = summary_df.sort_values(
        ["password_label", "number_of_workers"]
    ).reset_index(drop=True)

    metrics_df = metrics_df.sort_values(
        ["password_label", "number_of_workers", "metric"]
    ).reset_index(drop=True)

    summary_df = add_prediction_columns(summary_df)

    summary_df.to_csv(out_dir / "assignment_summary.csv", index=False)
    metrics_df.to_csv(out_dir / "assignment_metric_details.csv", index=False)

    save_required_measurements_table(
        summary_df,
        out_dir / "required_measurements_table.png",
    )

    create_per_experiment_scaling_charts(summary_df, out_dir)
    create_prediction_validation_chart(summary_df, out_dir)
    save_prediction_table(summary_df, out_dir / "prediction_table.png")

    print(f"Saved: {out_dir / 'assignment_summary.csv'}")
    print(f"Saved: {out_dir / 'assignment_metric_details.csv'}")
    print(f"Saved: {out_dir / 'required_measurements_table.png'}")
    print(f"Saved: per-experiment scaling charts in {out_dir}")

    if (out_dir / "prediction_table.png").exists():
        print(f"Saved: {out_dir / 'prediction_table.png'}")
    if (out_dir / "prediction_vs_measured_5_workers.png").exists():
        print(f"Saved: {out_dir / 'prediction_vs_measured_5_workers.png'}")

    print("\nPreview: assignment_summary.csv")
    print(summary_df.head(10).to_string(index=False))


if __name__ == "__main__":
    main()
