(defn msgs-success [msgs] (< msgs 30))
(defn mid-latency-success [lat] (< lat 400))
(defn max-latency-success [lat] (< lat 600))
(fn
  [data]
  (let [msgs-per-op (-> data :net :servers :msgs-per-op)
        median-latency (-> data :workload :stable-latencies (get 0.5))
        max-latency (-> data :workload :stable-latencies (get 1))]
    {:metrics {:messages-per-operation
               {:value msgs-per-op
                :less-than-30 (msgs-success msgs-per-op)}
               :median-latency
               {:value median-latency
                :less-than-400ms (mid-latency-success median-latency)}
               :maximum-latency
               {:value max-latency
                :less-than-600ms (max-latency-success max-latency)}}
     :success (every?
               true?
               [(msgs-success msgs-per-op)
                (mid-latency-success median-latency)
                (max-latency-success max-latency)])}))
